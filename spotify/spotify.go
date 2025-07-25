package spotify

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	golibrespot "github.com/devgianlu/go-librespot"
	"github.com/devgianlu/go-librespot/ap"
	"github.com/devgianlu/go-librespot/dealer"
	connectpb "github.com/devgianlu/go-librespot/proto/spotify/connectstate"
	devicespb "github.com/devgianlu/go-librespot/proto/spotify/connectstate/devices"
	metadatapb "github.com/devgianlu/go-librespot/proto/spotify/metadata"
	"github.com/devgianlu/go-librespot/session"
	"github.com/devgianlu/go-librespot/spclient"
	"github.com/go-co-op/gocron/v2"
	"github.com/sirupsen/logrus"

	"github.com/gregdel/pushover"
	"google.golang.org/protobuf/proto"

	"github.com/marcus-crane/gunslinger/config"
	"github.com/marcus-crane/gunslinger/db"
	"github.com/marcus-crane/gunslinger/playback"
	"github.com/marcus-crane/gunslinger/shared"
	"github.com/marcus-crane/gunslinger/utils"
)

const (
	authURL        = "https://accounts.spotify.com/authorize"
	tokenURL       = "https://accounts.spotify.com/api/token"
	accessTokenID  = "spotify:accesstoken"
	refreshTokenID = "spotify:refreshtoken"
)

type ProductInfo struct {
	XMLName  xml.Name `xml:"products"`
	Products []struct {
		XMLName      xml.Name `xml:"product"`
		Type         string   `xml:"type"`
		HeadFilesUrl string   `xml:"head-files-url"`
		ImageUrl     string   `xml:"image-url"`
		Autoplay     string   `xml:"autoplay"`
	} `xml:"product"`
}

func (pi ProductInfo) ImageUrl(fileId string) string {
	var imagePrefix string
	if len(pi.Products) != 0 {
		imagePrefix = pi.Products[0].ImageUrl
	} else {
		// If we boot up too fast or something, just fallback to what we assume the default value is
		imagePrefix = "https://i.scdn.co/image/{file_id}"
	}
	return strings.Replace(imagePrefix, "{file_id}", strings.ToLower(fileId), 1)
}

type Client struct {
	cfg          config.Config
	sess         *session.Session
	dealer       *dealer.Dealer
	sp           *spclient.Spclient
	prodInfo     *ProductInfo
	countryCode  string
	refreshToken string
	accessToken  string
	tokenExpiry  time.Time
	mu           sync.Mutex
}

func SetupSpotifyPoller(cfg config.Config, ps *playback.PlaybackSystem, store db.Store) {
	var client *Client
	var err error

	redirectUri := cfg.Spotify.RedirectUri

	if redirectUri == "" {
		panic("Empty redirect URI configured")
	}

	accessToken := store.GetTokenByID(accessTokenID)
	refreshToken := store.GetTokenByID(refreshTokenID)

	if accessToken == "" || refreshToken == "" {
		// Locally support using oauth instead of having a token (for server side)
		token, err := performOAuth2Flow(cfg, 8081)
		if err != nil {
			slog.With("error", err).Error("failed to generate access tokens")
		}
		// Save our newly generated token
		if err := store.UpsertToken(accessTokenID, token.AccessToken); err != nil {
			slog.With("error", err).Error("failed to save access token")
		}
		if err := store.UpsertToken(refreshTokenID, token.RefreshToken); err != nil {
			slog.With("error", err).Error("failed to save refresh token")
		}
		if err := store.UpsertTokenMetadata(accessTokenID, token.CreatedAt, token.ExpiresIn); err != nil {
			slog.With("error", err).Error("failed to save access token metadata")
		}
	}

	client, err = NewClient(cfg, store, accessToken, refreshToken)
	if err != nil {
		// we'll try oauth (which will ping my phone + let me auth remote) and see if we make it in time otherwise we'll need manual intervention
		// also this should be made less repetitive but whatever
		token, err := performOAuth2Flow(cfg, 8081)
		if err != nil {
			slog.With("error", err).Error("failed to perform oauth flow as fallback")
			return
		}
		if err := store.UpsertToken(accessTokenID, token.AccessToken); err != nil {
			slog.With("error", err).Error("failed to save access token")
		}
		if err := store.UpsertToken(refreshTokenID, token.RefreshToken); err != nil {
			slog.With("error", err).Error("failed to save refresh token")
		}
		if err := store.UpsertTokenMetadata(accessTokenID, token.CreatedAt, token.ExpiresIn); err != nil {
			slog.With("error", err).Error("failed to save access token metadata")
		}
		client, err = NewClient(cfg, store, accessToken, refreshToken)
		if err != nil {
			slog.With("error", err).Error("failed to create spotify client and fallback to oauth")
			return
		}
	}
	client.Run(ps)
}

func NewClient(cfg config.Config, store db.Store, accessToken, refreshToken string) (*Client, error) {
	// TODO: Create a slog adapter. This is only here to satisfy librespot's hard dependency on
	// a logger existing. Without it, getting a token will panic
	librespotLogger := &LogrusAdapter{logrus.NewEntry(logrus.StandardLogger())}
	opts := &session.Options{
		DeviceType: devicespb.DeviceType_SMARTWATCH,
		DeviceId:   cfg.Spotify.DeviceId,
		Log:        librespotLogger,
		Credentials: session.SpotifyTokenCredentials{
			Username: cfg.Spotify.Username, // You might need to fetch the username separately
			Token:    accessToken,
		},
	}

	sess, err := session.NewSessionFromOptions(context.Background(), opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}

	client := &Client{
		cfg:          cfg,
		sess:         sess,
		dealer:       sess.Dealer(),
		sp:           sess.Spclient(),
		accessToken:  accessToken,
		refreshToken: refreshToken,
		tokenExpiry:  time.Now().Add(time.Hour), // default is 1 hour
	}

	go client.tokenRefreshLoop(store)

	return client, nil
}

func performOAuth2Flow(cfg config.Config, port int) (*shared.TokenResponse, error) {
	state := shared.GenerateRandomString(16)
	ch := make(chan *shared.TokenResponse)
	var srv *http.Server

	pushoverApp := pushover.New(cfg.Pushover.Token)
	recipient := pushover.NewRecipient(cfg.Pushover.Recipient)

	// TODO: Ideally integrate with existing router to make life easier + support
	// possibly oauth refreshing server side (for bootstrapping)
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		token, err := exchangeCodeForToken(cfg, code)
		if err != nil {
			http.Error(w, "Error exchanging code for token", http.StatusInternalServerError)
			return
		}
		ch <- token
		fmt.Fprintf(w, "Authentication successful! You can close this window.")
		go func() {
			time.Sleep(time.Second)
			srv.Shutdown(r.Context())
		}()
	})

	srv = &http.Server{Addr: fmt.Sprintf(":%d", port)}
	go func() { _ = srv.ListenAndServe() }()

	// TODO: Extend when fetching initial state via API
	scopes := []string{
		"streaming",
	}

	url := fmt.Sprintf("%s?response_type=code&client_id=%s&scope=%s&redirect_uri=%s&state=%s",
		authURL, cfg.Spotify.ClientId, url.QueryEscape(strings.Join(scopes, " ")), url.QueryEscape(cfg.Spotify.RedirectUri), state)

	slog.With(slog.String("url", url)).Info("Please open the following URL in your browser")
	message := &pushover.Message{
		Message:    "Refresh token has expired + you probably redeployed so we need to manually reauth",
		Title:      "Please auth with Spotify for Gunslinger",
		Priority:   pushover.PriorityHigh,
		URL:        url,
		URLTitle:   "Auth with Spotify",
		Timestamp:  time.Now().Unix(),
		DeviceName: "Gunslinger",
	}
	_, err := pushoverApp.SendMessage(message, recipient)
	if err != nil {
		fmt.Println(err)
		return &shared.TokenResponse{}, fmt.Errorf("failed to notify about oauth request")
	}

	token := <-ch
	return token, nil
}

func exchangeCodeForToken(cfg config.Config, code string) (*shared.TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", cfg.Spotify.RedirectUri)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(cfg.Spotify.ClientId+":"+cfg.Spotify.ClientSecret)))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var token shared.TokenResponse
	err = json.Unmarshal(body, &token)
	if err != nil {
		return nil, err
	}

	return &token, nil
}

func (c *Client) refreshTokens() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", c.refreshToken)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.cfg.Spotify.ClientId+":"+c.cfg.Spotify.ClientSecret)))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var newTokens shared.TokenResponse
	err = json.Unmarshal(body, &newTokens)
	if err != nil {
		return err
	}

	c.accessToken = newTokens.AccessToken
	if newTokens.RefreshToken != "" {
		c.refreshToken = newTokens.RefreshToken
	}
	c.tokenExpiry = time.Now().Add(time.Duration(newTokens.ExpiresIn) * time.Second)

	slog.Info("Successfully refreshed tokens")

	return nil
}

func (c *Client) tokenRefreshLoop(store db.Store) {
	for {
		c.mu.Lock()
		timeUntilExpiry := time.Until(c.tokenExpiry)
		c.mu.Unlock()

		if timeUntilExpiry <= 5*time.Minute {
			refreshedTokens := false
			// TODO: Return shared tokens here and save results
			if err := c.refreshTokens(); err != nil {
				log.Printf("Failed to refresh token: %v", err)
				if err := c.reauthenticate(); err != nil {
					log.Printf("Failed to reauthenticate: %v", err)
				} else {
					refreshedTokens = true
				}
			} else {
				refreshedTokens = true
			}
			if refreshedTokens {
				// Refresh our new tokens so if we crash and restart, we should be good to go
				if err := store.UpsertToken(accessTokenID, c.accessToken); err != nil {
					slog.With("error", err).Error("failed to save access token")
				}
				if err := store.UpsertToken(refreshTokenID, c.refreshToken); err != nil {
					slog.With("error", err).Error("failed to save refresh token")
				}
			}
		}

		time.Sleep(1 * time.Minute)
	}
}

func (c *Client) reauthenticate() error {
	token, err := performOAuth2Flow(c.cfg, 8888)
	if err != nil {
		return fmt.Errorf("failed to reauthenticate: %v", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.accessToken = token.AccessToken
	c.refreshToken = token.RefreshToken
	c.tokenExpiry = time.Now().Add(time.Second * time.Duration(token.ExpiresIn))

	opts := &session.Options{
		DeviceType: devicespb.DeviceType_SMARTWATCH,
		DeviceId:   c.cfg.Spotify.DeviceId,
		Credentials: session.SpotifyTokenCredentials{
			Username: c.cfg.Spotify.Username,
			Token:    c.accessToken,
		},
	}

	newSess, err := session.NewSessionFromOptions(context.TODO(), opts)
	if err != nil {
		return fmt.Errorf("failed to create new session: %v", err)
	}

	c.sess.Close()
	c.sess = newSess
	c.dealer = newSess.Dealer()
	c.sp = newSess.Spclient()

	slog.Info("Successfully recreated session with refreshed tokens")

	return nil
}

func (c *Client) Run(ps *playback.PlaybackSystem) {
	if err := c.sess.Dealer().Connect(context.TODO()); err != nil {
		slog.With(slog.String("error", err.Error())).Error("Failed to connect to dealer")
	}
	apRecv := c.sess.Accesspoint().Receive(ap.PacketTypeProductInfo, ap.PacketTypeCountryCode)
	msgChan := c.dealer.ReceiveMessage("hm://pusher/v1/connections/", "hm://connect-state/v1/")
	reqRecv := c.sess.Dealer().ReceiveRequest("hm://connect-state/v1/player/command")

	// Not sure if there is a way to poll Spotify for live state so instead we'll just
	// fake it by incrementing in the background. Could do this client side but less
	// futzing with Javascript if we just do it this side and kinda nicer API experience.
	s, _ := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	s.NewJob(
		gocron.DurationJob(time.Second*5),
		gocron.NewTask(c.rehydratePlaybackState, ps),
	)

	s.Start()

	for {
		select {
		case pkt := <-apRecv:
			c.handleAccessPointPacket(pkt.Type, pkt.Payload)
		case msg := <-msgChan:
			c.handleMessage(msg, ps)
		case req := <-reqRecv:
			c.handleDealerRequest(req, ps)
		}
	}
}

func (c *Client) rehydratePlaybackState(ps *playback.PlaybackSystem) {
	playbackState, err := ps.GetActivePlaybackBySource(string(playback.Spotify))
	if err != nil {
		return
	}
	if len(playbackState) != 1 {
		// Shouldn't be possible but not sure what to do here
		return
	}
	state := playbackState[0]
	if state.Status != playback.StatusPlaying {
		slog.Debug("rehydration: status was not playing so skipping")
		return
	}
	if state.CreatedAt.After(time.Now()) {
		slog.Debug("rehydration: created at time was after now")
		// should be impossible but we'll bail out since time travel is impossible
		return
	}
	projectedElapsed := time.Now().Sub(state.CreatedAt)
	projectedElapsedMs := int(projectedElapsed.Milliseconds())
	if projectedElapsedMs >= state.Duration {
		slog.Debug("rehydration: elapsed projected would be equal or after duration",
			slog.Int("totalElapsed", projectedElapsedMs),
			slog.Int("duration", state.Duration))
		// we seem to have surpassed the end of the song so do nothing
		return
	}
	fakeUpdate := playback.Update{
		MediaItem: playback.MediaItem{
			Title:           state.Title,
			Subtitle:        state.Subtitle,
			Category:        state.Category,
			Duration:        state.Duration,
			Source:          state.Source,
			Image:           state.Image,
			DominantColours: state.DominantColours,
		},
		Elapsed: projectedElapsed,
		Status:  state.Status,
	}
	if err := ps.UpdatePlaybackState(fakeUpdate); err != nil {
		slog.Error("Failed to save Spotify update",
			slog.String("error", err.Error()),
			slog.String("title", fakeUpdate.MediaItem.Title))
	}
}

func (c *Client) handleDealerRequest(req dealer.Request, ps *playback.PlaybackSystem) {
	slog.With("uri", req.MessageIdent).Info("received request from spotify")
	switch req.MessageIdent {
	case "hm://connect-state/v1/player/command":
		c.handlePlayerCommand(req.Payload, ps)
	default:
		slog.With(slog.String("ident", req.MessageIdent)).Warn("unknown dealer request: %s")
	}
}

func (c *Client) handlePlayerCommand(req dealer.RequestPayload, _ *playback.PlaybackSystem) {
	switch req.Command.Endpoint {
	case "transfer":
		var transferState connectpb.TransferState
		if err := proto.Unmarshal(req.Command.Data, &transferState); err != nil {
			slog.With(slog.String("error", err.Error())).Error("failed unmarshalling TransferState")
		}
		slog.With(slog.String("command", "transfer")).Info(transferState.GetPlayback().GetCurrentTrack().String())
	default:
		slog.With(slog.String("endpoint", req.Command.Endpoint)).Error("unsupported player command")
	}
}

func (c *Client) handleAccessPointPacket(pktType ap.PacketType, payload []byte) error {
	switch pktType {
	case ap.PacketTypeProductInfo:
		var prod ProductInfo
		if err := xml.Unmarshal(payload, &prod); err != nil {
			return fmt.Errorf("failed umarshalling ProductInfo: %w", err)
		}

		if len(prod.Products) != 1 {
			return fmt.Errorf("invalid ProductInfo")
		}

		c.prodInfo = &prod
		return nil
	case ap.PacketTypeCountryCode:
		c.countryCode = string(payload)
		return nil
	default:
		return nil
	}
}

func (c *Client) handleMessage(msg dealer.Message, ps *playback.PlaybackSystem) {
	slog.With("uri", msg.Uri).Info("received message from spotify")
	if strings.HasPrefix(msg.Uri, "hm://pusher/v1/connections/") {
		spotConnId := msg.Headers["Spotify-Connection-Id"]
		slog.With(slog.String("connection_id", spotConnId)).Debug("Established connection to Spotify")
		// TODO: Generate own client ID
		clientId := hex.EncodeToString([]byte{0x65, 0xb7, 0x8, 0x7, 0x3f, 0xc0, 0x48, 0xe, 0xa9, 0x2a, 0x7, 0x72, 0x33, 0xca, 0x87, 0xbd})
		putStateReq := &connectpb.PutStateRequest{
			ClientSideTimestamp: uint64(time.Now().UnixMilli()),
			MemberType:          connectpb.MemberType_CONNECT_STATE,
			PutStateReason:      connectpb.PutStateReason_NEW_DEVICE,
			Device: &connectpb.Device{
				DeviceInfo: &connectpb.DeviceInfo{
					Name:                  c.cfg.Spotify.ConnectPlayerName,
					Volume:                0,
					CanPlay:               false,
					DeviceType:            devicespb.DeviceType_SMARTWATCH,
					DeviceSoftwareVersion: "gunslinger 1.0.0",
					ClientId:              clientId,
					Brand:                 "utf9k",
					Model:                 "Gunslinger",
					SpircVersion:          "1.0.0",
					Capabilities: &connectpb.Capabilities{
						CanBePlayer:  false,
						IsObservable: true,
						// TODO: Type for audiobooks(?)
						SupportedTypes:          []string{"audio/track", "audio/episode"},
						CommandAcks:             true,
						DisableVolume:           true,
						ConnectDisabled:         false,
						SupportsPlaylistV2:      true,
						Hidden:                  false,
						NeedsFullPlayerState:    false,
						SupportsTransferCommand: false,
						SupportsCommandRequest:  true,
						SupportsGzipPushes:      true,
						ConnectCapabilities:     "",
					},
				},
			},
		}
		c.sp.PutConnectState(context.TODO(), spotConnId, putStateReq)
	}
	// TODO: Support initialising state as we only get a cluster update after an action happens
	if strings.HasPrefix(msg.Uri, "hm://connect-state/v1/cluster") {
		var clusterUpdate connectpb.ClusterUpdate
		if err := proto.Unmarshal(msg.Payload, &clusterUpdate); err != nil {
			fmt.Printf("failed to unmarshal cluster update %+v", err)
			return
		}
		// TODO: Check if newer versions of golibrespot handle this
		if strings.Contains(clusterUpdate.Cluster.PlayerState.Track.Uri, ":ad:") {
			// This is an ad and we can't do anything with it. In 0.0.18 at least, SpotifyIdFromUri will crash
			return
		}
		spotifyId, err := golibrespot.SpotifyIdFromUri(clusterUpdate.Cluster.PlayerState.Track.Uri)
		if err != nil {
			return
		}

		status := playback.StatusPlaying

		if clusterUpdate.Cluster.PlayerState.IsPaused {
			status = playback.StatusPaused
		}

		update := playback.Update{
			MediaItem: playback.MediaItem{
				Duration: int(clusterUpdate.Cluster.PlayerState.GetDuration()),
				Source:   string(playback.Spotify),
			},
			// nanoseconds -> seconds
			Elapsed: time.Duration(clusterUpdate.Cluster.PlayerState.GetPositionAsOfTimestamp() * 1000000),
			Status:  status,
		}

		var coverId string

		if spotifyId.Type() == golibrespot.SpotifyIdTypeTrack {
			track, err := c.sp.MetadataForTrack(context.TODO(), *spotifyId)
			if err != nil {
				return
			}

			update.MediaItem.Title = track.GetName()
			update.MediaItem.Subtitle = track.GetArtist()[0].GetName()
			update.MediaItem.Category = "track"

			coverId = pullAlbumCoverId(track)
		} else if spotifyId.Type() == golibrespot.SpotifyIdTypeEpisode {
			episode, err := c.sp.MetadataForEpisode(context.TODO(), *spotifyId)
			if err != nil {
				return
			}

			update.MediaItem.Title = episode.GetName()
			update.MediaItem.Subtitle = episode.GetShow().GetName()
			update.MediaItem.Category = "podcast_episode"

			coverId = pullEpisodeCoverId(episode)
		} else {
			// no idea what type this is
			return
		}

		// We know this will be something so we'll deactivate all other Spotify
		// playbacks and resume them again. It might even be this one but that's ok.
		ps.DeactivateBySource(string(playback.Spotify))

		imageUrl := c.prodInfo.ImageUrl(coverId)
		image, extension, domColours, err := utils.ExtractImageContent(imageUrl)
		if err != nil {
			slog.Error("Failed to extract image content",
				slog.String("error", err.Error()),
				slog.String("image_url", imageUrl),
			)
			return
		}

		update.MediaItem.DominantColours = domColours

		hash := playback.GenerateMediaID(&update)
		coverUrl, err := utils.SaveCover(c.cfg, hash, image, extension)
		if err != nil {
			slog.Error("Failed to save cover for Spotify",
				slog.String("error", err.Error()),
				slog.String("guid", hash),
				slog.String("title", update.MediaItem.Title),
			)
		}

		update.MediaItem.Image = coverUrl

		if err := ps.UpdatePlaybackState(update); err != nil {
			slog.Error("Failed to save Spotify update",
				slog.String("error", err.Error()),
				slog.String("title", update.MediaItem.Title))
		}
	}
}

func pullAlbumCoverId(track *metadatapb.Track) string {
	var albumCoverId []byte
	if len(track.Album.Cover) > 0 {
		coverId := track.Album.Cover[0].FileId
		for _, c := range track.Album.Cover {
			if c.GetSize() == metadatapb.Image_LARGE {
				coverId = c.FileId
			}
		}
		albumCoverId = coverId
	} else if track.Album.CoverGroup != nil && len(track.Album.CoverGroup.Image) > 0 {
		coverId := track.Album.CoverGroup.Image[0].FileId
		for _, c := range track.Album.CoverGroup.Image {
			if c.GetSize() == metadatapb.Image_LARGE {
				coverId = c.FileId
			}
		}
		albumCoverId = coverId
	}
	return hex.EncodeToString(albumCoverId)
}

func pullEpisodeCoverId(episode *metadatapb.Episode) string {
	var episodeCoverId []byte
	if len(episode.GetShow().GetCoverImage().GetImage()) > 0 {
		coverId := episode.GetShow().GetCoverImage().GetImage()[0].FileId
		for _, c := range episode.GetShow().GetCoverImage().GetImage() {
			if c.GetSize() == metadatapb.Image_LARGE {
				coverId = c.FileId
			}
		}
		episodeCoverId = coverId
	}
	if len(episode.GetCoverImage().GetImage()) > 0 {
		coverId := episode.GetCoverImage().GetImage()[0].FileId
		for _, c := range episode.GetCoverImage().GetImage() {
			if c.GetSize() == metadatapb.Image_LARGE {
				coverId = c.FileId
			}
		}
		episodeCoverId = coverId
	}
	return hex.EncodeToString(episodeCoverId)
}

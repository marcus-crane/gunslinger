package spotify

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	golibrespot "github.com/devgianlu/go-librespot"
	"github.com/devgianlu/go-librespot/ap"
	"github.com/devgianlu/go-librespot/dealer"
	"github.com/devgianlu/go-librespot/login5"
	connectpb "github.com/devgianlu/go-librespot/proto/spotify/connectstate"
	extmetadatapb "github.com/devgianlu/go-librespot/proto/spotify/extendedmetadata"
	devicespb "github.com/devgianlu/go-librespot/proto/spotify/connectstate/devices"
	metadatapb "github.com/devgianlu/go-librespot/proto/spotify/metadata"
	"github.com/devgianlu/go-librespot/session"
	"github.com/devgianlu/go-librespot/spclient"
	"github.com/go-co-op/gocron/v2"

	"github.com/gregdel/pushover"
	"google.golang.org/protobuf/proto"

	"github.com/marcus-crane/gunslinger/config"
	"github.com/marcus-crane/gunslinger/db"
	"github.com/marcus-crane/gunslinger/playback"
)

const (
	usernameID          = "spotify:username"
	storedCredentialsID = "spotify:storedcredentials"
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

func (pi *ProductInfo) ImageUrl(fileId string) string {
	// If we boot up too fast or something, just fallback to what we assume the default value is
	imagePrefix := "https://i.scdn.co/image/{file_id}"
	if pi != nil && len(pi.Products) != 0 {
		imagePrefix = pi.Products[0].ImageUrl
	}
	return strings.Replace(imagePrefix, "{file_id}", strings.ToLower(fileId), 1)
}

type Client struct {
	cfg         config.Config
	store       db.Store
	sess        *session.Session
	dealer      *dealer.Dealer
	sp          *spclient.Spclient
	prodInfo    *ProductInfo
	countryCode string
	mu          sync.Mutex
}

func SetupSpotifyPoller(cfg config.Config, ps *playback.PlaybackSystem, store db.Store) {
	client := &Client{cfg: cfg, store: store}
	sess, err := client.newSession()
	if err != nil {
		slog.With("error", err).Error("failed to create spotify session")
		return
	}
	client.sess = sess
	client.dealer = sess.Dealer()
	client.sp = sess.Spclient()
	client.Run(ps)
}

// newSession logs in with the credentials saved from the last session. If
// Spotify rejects them, or none are saved, it falls back to pairing this
// device, which sends a link to approve the login over Pushover and waits
// until it is approved or the code expires.
func (c *Client) newSession() (*session.Session, error) {
	username := c.store.GetTokenByID(usernameID)
	stored := c.store.GetTokenByID(storedCredentialsID)
	if username != "" && stored != "" {
		data, err := base64.StdEncoding.DecodeString(stored)
		if err != nil {
			return nil, fmt.Errorf("failed decoding saved credentials: %w", err)
		}
		sess, err := c.connect(session.StoredCredentials{Username: username, Data: data})
		if err == nil {
			c.saveCredentials(sess)
			return sess, nil
		}
		if !credentialsRejected(err) {
			return nil, err
		}
		slog.With("error", err).Warn("Spotify rejected the saved credentials so pairing again")
	}

	sess, err := c.connect(session.DeviceAuthCredentials{OnCode: c.sendPairingCode})
	if err != nil {
		return nil, err
	}
	c.saveCredentials(sess)
	return sess, nil
}

func (c *Client) connect(creds any) (*session.Session, error) {
	opts := &session.Options{
		DeviceType:  devicespb.DeviceType_SMARTWATCH,
		DeviceId:    c.cfg.Spotify.DeviceId,
		Log:         &SlogAdapter{slog.Default()},
		Credentials: creds,
	}
	sess, err := session.NewSessionFromOptions(context.Background(), opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	return sess, nil
}

// credentialsRejected separates Spotify refusing the credentials from network
// failures, so that a blip while rebuilding a session doesn't start pairing.
func credentialsRejected(err error) bool {
	var apErr *ap.AccesspointLoginError
	var login5Err *login5.LoginError
	return errors.As(err, &apErr) || errors.As(err, &login5Err)
}

func (c *Client) saveCredentials(sess *session.Session) {
	if err := c.store.UpsertToken(usernameID, sess.Username()); err != nil {
		slog.With("error", err).Error("failed to save spotify username")
	}
	stored := base64.StdEncoding.EncodeToString(sess.StoredCredentials())
	if err := c.store.UpsertToken(storedCredentialsID, stored); err != nil {
		slog.With("error", err).Error("failed to save spotify credentials")
	}
}

func (c *Client) sendPairingCode(code *session.DeviceAuthCode) {
	// The session calls this again with nil once the code is used or expires.
	if code == nil {
		return
	}
	message := &pushover.Message{
		Message:    fmt.Sprintf("Approve the login and enter %s if asked. The code expires at %s.", code.UserCode, code.ExpiresAt.Format(time.Kitchen)),
		Title:      "Please auth with Spotify for Gunslinger",
		Priority:   pushover.PriorityHigh,
		URL:        code.VerificationUrl,
		URLTitle:   "Auth with Spotify",
		Timestamp:  time.Now().Unix(),
		DeviceName: "Gunslinger",
	}
	recipient := pushover.NewRecipient(c.cfg.Pushover.Recipient)
	if _, err := pushover.New(c.cfg.Pushover.Token).SendMessage(message, recipient); err != nil {
		slog.With("error", err).Error("failed to send pushover notification for spotify auth")
	}
}

func (c *Client) Run(ps *playback.PlaybackSystem) {
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
		c.mu.Lock()
		sess := c.sess
		dlr := c.dealer
		c.mu.Unlock()

		if err := dlr.Connect(context.TODO()); err != nil {
			slog.With(slog.String("error", err.Error())).Error("Failed to connect to dealer")
			time.Sleep(time.Minute)
			continue
		}

		apRecv := sess.Accesspoint().Receive(ap.PacketTypeProductInfo, ap.PacketTypeCountryCode)
		msgChan := dlr.ReceiveMessage("hm://pusher/v1/connections/", "hm://connect-state/v1/")
		reqRecv := dlr.ReceiveRequest("hm://connect-state/v1/player/command")

	recv:
		for {
			select {
			case pkt, ok := <-apRecv:
				if !ok {
					break recv
				}
				c.handleAccessPointPacket(pkt.Type, pkt.Payload)
			case msg, ok := <-msgChan:
				if !ok {
					break recv
				}
				c.handleMessage(msg, ps)
			case req, ok := <-reqRecv:
				if !ok {
					break recv
				}
				c.handleDealerRequest(req, ps)
			}
		}

		slog.Warn("Spotify dealer channels closed, moving to a fresh session")
		for {
			c.mu.Lock()
			replaced := c.dealer != dlr
			c.mu.Unlock()
			if replaced {
				break
			}
			if err := c.rebuildSession(); err != nil {
				slog.Error("Failed to rebuild Spotify session", slog.String("error", err.Error()))
				time.Sleep(time.Minute)
				continue
			}
			break
		}
	}
}

func (c *Client) rebuildSession() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	newSess, err := c.newSession()
	if err != nil {
		return err
	}

	oldSess := c.sess
	go oldSess.Close()

	c.sess = newSess
	c.dealer = newSess.Dealer()
	c.sp = newSess.Spclient()

	return nil
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
		slog.Warn("unknown dealer request", slog.String("ident", req.MessageIdent))
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
		if _, err := c.sp.PutConnectState(context.TODO(), spotConnId, putStateReq); err != nil {
			slog.Error("Failed to put connect state",
				slog.String("error", err.Error()),
				slog.String("connection_id", spotConnId))
		}
	}
	// TODO: Support initialising state as we only get a cluster update after an action happens
	if strings.HasPrefix(msg.Uri, "hm://connect-state/v1/cluster") {
		var clusterUpdate connectpb.ClusterUpdate
		if err := proto.Unmarshal(msg.Payload, &clusterUpdate); err != nil {
			slog.Error("Failed to unmarshal cluster update", slog.String("error", err.Error()))
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
			var track metadatapb.Track
			if err := c.sp.ExtendedMetadataSimple(context.TODO(), *spotifyId, extmetadatapb.ExtensionKind_TRACK_V4, &track); err != nil {
				return
			}

			update.MediaItem.Title = track.GetName()
			update.MediaItem.Subtitle = track.GetArtist()[0].GetName()
			update.MediaItem.Category = "track"

			coverId = pullAlbumCoverId(&track)
		} else if spotifyId.Type() == golibrespot.SpotifyIdTypeEpisode {
			var episode metadatapb.Episode
			if err := c.sp.ExtendedMetadataSimple(context.TODO(), *spotifyId, extmetadatapb.ExtensionKind_EPISODE_V4, &episode); err != nil {
				return
			}

			update.MediaItem.Title = episode.GetName()
			update.MediaItem.Subtitle = episode.GetShow().GetName()
			update.MediaItem.Category = "podcast_episode"

			coverId = pullEpisodeCoverId(&episode)
		} else {
			// no idea what type this is
			return
		}

		// We know this will be something so we'll deactivate all other Spotify
		// playbacks and resume them again. It might even be this one but that's ok.
		ps.DeactivateBySource(string(playback.Spotify))

		hash := playback.GenerateMediaID(&update)
		coverUrl, domColours, err := ps.ResolveCover(c.cfg, hash, c.prodInfo.ImageUrl(coverId))
		if err != nil {
			slog.Error("Failed to resolve cover for Spotify",
				slog.String("error", err.Error()),
				slog.String("title", update.MediaItem.Title),
			)
			return
		}
		update.MediaItem.Image = coverUrl
		update.MediaItem.DominantColours = domColours

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

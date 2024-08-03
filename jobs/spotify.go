package jobs

import (
	"bytes"
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
	"os"
	"strings"
	"sync"
	"time"

	golibrespot "go-librespot"
	"go-librespot/ap"
	"go-librespot/dealer"
	connectpb "go-librespot/proto/spotify/connectstate"
	devicespb "go-librespot/proto/spotify/connectstate/devices"
	metadatapb "go-librespot/proto/spotify/metadata"
	"go-librespot/session"
	"go-librespot/spclient"

	"github.com/r3labs/sse/v2"
	"golang.org/x/exp/rand"
	"google.golang.org/protobuf/proto"

	"github.com/marcus-crane/gunslinger/db"
	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/models"
	"github.com/marcus-crane/gunslinger/utils"
)

const (
	redirectURI = "http://localhost:8081/callback"
	authURL     = "https://accounts.spotify.com/authorize"
	tokenURL    = "https://accounts.spotify.com/api/token"
)

var (
	accessToken  = os.Getenv("SPOTIFY_ACCESS_TOKEN")
	refreshToken = os.Getenv("SPOTIFY_REFRESH_TOKEN")
	deviceId     = utils.MustEnv("SPOTIFY_DEVICE_ID")
	clientID     = utils.MustEnv("SPOTIFY_CLIENT_ID")
	clientSecret = utils.MustEnv("SPOTIFY_CLIENT_SECRET")
	username     = utils.MustEnv("SPOTIFY_USERNAME")
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

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

func SetupSpotifyPoller(store db.Store) {
	var err error

	if accessToken == "" || refreshToken == "" {
		// Locally support using oauth instead of having a token (for server side)
		accessToken, refreshToken, err = performOAuth2Flow(8081)
		if err != nil {
			panic(err)
		}
	}

	client, err := NewClient(deviceId, accessToken, refreshToken)
	if err != nil {
		panic(err)
	}
	client.Run(store)
}

func NewClient(deviceId, accessToken, refreshToken string) (*Client, error) {

	opts := &session.Options{
		DeviceType: devicespb.DeviceType_SMARTWATCH,
		DeviceId:   deviceId,
		Credentials: session.SpotifyTokenCredentials{
			Username: username, // You might need to fetch the username separately
			Token:    accessToken,
		},
	}

	sess, err := session.NewSessionFromOptions(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}

	client := &Client{
		sess:         sess,
		dealer:       sess.Dealer(),
		sp:           sess.Spclient(),
		accessToken:  accessToken,
		refreshToken: refreshToken,
		tokenExpiry:  time.Now().Add(time.Hour), // default is 1 hour
	}

	go client.tokenRefreshLoop()

	return client, nil
}

func performOAuth2Flow(port int) (string, string, error) {
	state := generateRandomString(16)
	ch := make(chan *TokenResponse)
	var srv *http.Server

	// TODO: Ideally integrate with existing router to make life easier + support
	// possibly oauth refreshing server side (for bootstrapping)
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		token, err := exchangeCodeForToken(code)
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
		authURL, clientID, url.QueryEscape(strings.Join(scopes, " ")), url.QueryEscape(redirectURI), state)

	slog.With(slog.String("url", url)).Info("Please open the following URL in your browser")

	token := <-ch
	return token.AccessToken, token.RefreshToken, nil
}

func exchangeCodeForToken(code string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(clientID+":"+clientSecret)))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var token TokenResponse
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
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(clientID+":"+clientSecret)))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var newTokens TokenResponse
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

func (c *Client) tokenRefreshLoop() {
	for {
		c.mu.Lock()
		timeUntilExpiry := time.Until(c.tokenExpiry)
		c.mu.Unlock()

		if timeUntilExpiry <= 5*time.Minute {
			if err := c.refreshTokens(); err != nil {
				log.Printf("Failed to refresh token: %v", err)
				if err := c.reauthenticate(); err != nil {
					log.Printf("Failed to reauthenticate: %v", err)
				}
			}
		}

		time.Sleep(1 * time.Minute)
	}
}

func (c *Client) reauthenticate() error {
	accessToken, refreshToken, err := performOAuth2Flow(8888)
	if err != nil {
		return fmt.Errorf("failed to reauthenticate: %v", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.accessToken = accessToken
	c.refreshToken = refreshToken
	c.tokenExpiry = time.Now().Add(time.Hour) // Assume 1 hour expiry, adjust as needed

	opts := &session.Options{
		DeviceType: devicespb.DeviceType_SMARTWATCH,
		DeviceId:   deviceId,
		Credentials: session.SpotifyTokenCredentials{
			Username: username,
			Token:    c.accessToken,
		},
	}

	newSess, err := session.NewSessionFromOptions(opts)
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

func (c *Client) Run(store db.Store) {
	apRecv := c.sess.Accesspoint().Receive(ap.PacketTypeProductInfo, ap.PacketTypeCountryCode)
	msgChan := c.dealer.ReceiveMessage("hm://pusher/v1/connections/", "hm://connect-state/v1/")

	for {
		select {
		case pkt := <-apRecv:
			c.handleAccessPointPacket(pkt.Type, pkt.Payload)
		case msg := <-msgChan:
			c.handleMessage(msg, store)
		}
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

func (c *Client) handleMessage(msg dealer.Message, store db.Store) {
	if strings.HasPrefix(msg.Uri, "hm://pusher/v1/connections/") {
		spotConnId := msg.Headers["Spotify-Connection-Id"]
		slog.With(slog.String("connection_id", spotConnId)).Info("Established connection to Spotify")
		// TODO: Generate own client ID
		clientId := hex.EncodeToString([]byte{0x65, 0xb7, 0x8, 0x7, 0x3f, 0xc0, 0x48, 0xe, 0xa9, 0x2a, 0x7, 0x72, 0x33, 0xca, 0x87, 0xbd})
		putStateReq := &connectpb.PutStateRequest{
			ClientSideTimestamp: uint64(time.Now().UnixMilli()),
			MemberType:          connectpb.MemberType_CONNECT_STATE,
			PutStateReason:      connectpb.PutStateReason_NEW_DEVICE,
			Device: &connectpb.Device{
				DeviceInfo: &connectpb.DeviceInfo{
					Name:                  "Gunslinger",
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
						Hidden:                  false,
						NeedsFullPlayerState:    false,
						SupportsTransferCommand: false,
						ConnectCapabilities:     "",
					},
				},
			},
		}
		c.sp.PutConnectState(spotConnId, putStateReq)
	}
	// TODO: Support initialising state as we only get a cluster update after an action happens
	if strings.HasPrefix(msg.Uri, "hm://connect-state/v1/cluster") {
		var clusterUpdate connectpb.ClusterUpdate
		if err := proto.Unmarshal(msg.Payload, &clusterUpdate); err != nil {
			fmt.Printf("failed to unmarshal cluster update %+v", err)
			return
		}
		spotifyTrackId := golibrespot.SpotifyIdFromUri(clusterUpdate.Cluster.PlayerState.Track.Uri)
		track, err := c.sp.MetadataForTrack(spotifyTrackId)
		if err != nil {
			return
		}

		albumCoverId := pullAlbumCoverId(track)
		imageUrl := c.prodInfo.ImageUrl(albumCoverId)

		image, extension, domColours, err := utils.ExtractImageContent(imageUrl)
		if err != nil {
			slog.Error("Failed to extract image content",
				slog.String("stack", err.Error()),
				slog.String("image_url", imageUrl),
			)
			return
		}
		imageLocation, guid := utils.BytesToGUIDLocation(image, extension)

		// TODO: Support podcasts and audiobooks
		playingItem := models.MediaItem{
			CreatedAt:       time.Now().Unix(),
			Title:           track.GetName(),
			Subtitle:        track.GetArtist()[0].GetName(),
			IsActive:        !clusterUpdate.Cluster.PlayerState.IsPaused,
			Category:        "track",
			Elapsed:         int(clusterUpdate.Cluster.PlayerState.GetPositionAsOfTimestamp()),
			Duration:        int(clusterUpdate.Cluster.PlayerState.GetDuration()),
			Source:          "spotify",
			DominantColours: domColours,
			Image:           imageLocation,
		}

		if CurrentPlaybackItem.GenerateHash() != playingItem.GenerateHash() {
			byteStream := new(bytes.Buffer)
			json.NewEncoder(byteStream).Encode(playingItem)
			events.Server.Publish("playback", &sse.Event{Data: byteStream.Bytes()})
			// We want to make sure that we don't resave if the server restarts
			// to ensure the history endpoint is relatively accurate
			previousItem, err := store.GetByCategory(playingItem.Category)
			if err == nil || err.Error() == "sql: no rows in result set" {
				if CurrentPlaybackItem.Title != playingItem.Title && previousItem.Title != playingItem.Title {
					if err := saveCover(guid.String(), image, extension); err != nil {
						slog.Error("Failed to save cover for Plex",
							slog.String("stack", err.Error()),
							slog.String("guid", guid.String()),
							slog.String("title", playingItem.Title),
						)
					}
					if err := store.Insert(playingItem); err != nil {
						slog.Error("Failed to save DB entry",
							slog.String("stack", err.Error()),
							slog.String("title", playingItem.Title),
						)
					}
				}
			} else {
				slog.Error("An unknown error occurred",
					slog.String("stack", err.Error()),
					slog.String("title", playingItem.Title),
				)
			}
		}
		CurrentPlaybackItem = playingItem
	}
}

func generateRandomString(length int) string {
	rand.Seed(uint64(time.Now().UnixNano()))
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
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

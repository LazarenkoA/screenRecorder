package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/pion/sdp/v3"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"screenRecorder/screenshot"
)

type newSessionRequest struct {
	Offer  string `json:"offer"`
	Screen int    `json:"screen"`
}

type newSessionResponse struct {
	Answer string `json:"answer"`
}

func main() {
	fps := 90
	screens := screen.InitDisplays(fps)

	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	result := make([]chan []byte, len(screens))
	stream := make(chan []byte)
	for i, s := range screens {
		result[i] = s.StartRecord(ctx)
	}

	file, err := os.Create("example.264")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer file.Close()

	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "%s\n", err.Error())
	// 	os.Exit(1)
	// }
	// defer enc.Close()

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	defer func() {
		if cErr := peerConnection.Close(); cErr != nil {
			fmt.Printf("cannot close peerConnection: %v\n", cErr)
		}
	}()

	// track, err := peerConnection.NewTrack(
	// 	webrtcCodec.PayloadType,
	// 	uint32(rand.Int31()),
	// 	uuid.New().String(),
	// 	fmt.Sprintf("remote-screen"),
	// )

	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video", "pion")
	if err != nil {
		panic(err)
	}

	peerConnection.OnICEConnectionStateChange(func(connState webrtc.ICEConnectionState) {
		if connState == webrtc.ICEConnectionStateConnected {
			go func() {
				for img := range stream {
					if err = videoTrack.WriteSample(media.Sample{Data: img, Duration: time.Second}); err != nil {
						fmt.Println(err)
					}
				}
			}()
		}
		if connState == webrtc.ICEConnectionStateDisconnected {
			fmt.Println("Disconnected")
		}
		log.Printf("Connection state: %s \n", connState.String())
	})

	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", makeHandler(videoTrack, peerConnection)))
	mux.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("./web"))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		http.ServeFile(w, r, "./web/index.html")
	})

	go func() {
		err := http.ListenAndServe(":8080", mux)
		if err != nil {
			fmt.Println(err)
		}
	}()

	var offset int
	for img := range result[0] {
		stream <- img

		file.WriteAt(img, int64(offset))
		offset += len(img)
		// fmt.Println(offset, err)

		// enc.Encode(img)
	}
}

func getImageFromFilePath(filePath string) image.Image {
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()
	image, _, err := image.Decode(f)
	return image
}

func makeHandler(videoTrack *webrtc.TrackLocalStaticSample, peerConn *webrtc.PeerConnection) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()

		offer := newSessionRequest{}
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &offer)

		sdp := sdp.SessionDescription{}
		err := sdp.Unmarshal([]byte(offer.Offer))

		if direction := getTrackDirection(&sdp); direction == webrtc.RTPTransceiverDirectionSendrecv {
			_, err = peerConn.AddTrack(videoTrack)
		} else if direction == webrtc.RTPTransceiverDirectionRecvonly {
			_, err = peerConn.AddTransceiverFromTrack(videoTrack, webrtc.RtpTransceiverInit{
				Direction: webrtc.RTPTransceiverDirectionSendonly,
			})
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		offerSdp := webrtc.SessionDescription{
			SDP:  offer.Offer,
			Type: webrtc.SDPTypeOffer,
		}
		if err := peerConn.SetRemoteDescription(offerSdp); err != nil {
			panic(err)
		}

		answer, err := peerConn.CreateAnswer(nil)
		if err != nil {
			panic(err)
		}
		peerConn.SetLocalDescription(answer)
		payload, err := json.Marshal(newSessionResponse{
			Answer: answer.SDP,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(payload)
	})

	return mux
}

func createAnswer() string {
	return ""
}

func getTrackDirection(sdp *sdp.SessionDescription) webrtc.RTPTransceiverDirection {
	for _, mediaDesc := range sdp.MediaDescriptions {
		if mediaDesc.MediaName.Media == "video" {
			if _, recvOnly := mediaDesc.Attribute("recvonly"); recvOnly {
				return webrtc.RTPTransceiverDirectionRecvonly
			} else if _, sendRecv := mediaDesc.Attribute("sendrecv"); sendRecv {
				return webrtc.RTPTransceiverDirectionSendrecv
			}
		}
	}
	return webrtc.RTPTransceiverDirectionInactive
}

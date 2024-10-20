package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/cors"
	qrcode "github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func main () {

	// |------------------------------------------------------------------------------------------------------|
	// | NOTE: You must also import the appropriate DB connector, e.g. github.com/mattn/go-sqlite3 for SQLite |
	// |------------------------------------------------------------------------------------------------------|

	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New("sqlite3", "file:sessions.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	clientLogger := waLog.Stdout("Client", "DEBUG", true);

	client := whatsmeow.NewClient(deviceStore, clientLogger)

	mux := http.NewServeMux()

	helloHandler := func (res http.ResponseWriter, req *http.Request) {
		io.WriteString(res, "Hello, world!! \n")
	}

	connectWithClient := func (res http.ResponseWriter, req *http.Request) {
		if (client.Store.ID == nil) {
			return
		}

		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
	}

	generateQr := func (res http.ResponseWriter, req *http.Request) {

		if client.Store.ID != nil {
			return
		}

		qrChan, _ := client.GetQRChannel(context.Background());
		err := client.Connect();

		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
				// e.g. qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal

				qr, err := qrcode.New(evt.Code, qrcode.Medium);
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to generate QR code: %v\n", err)
					return
				}

				// Get the QR code as a 2D boolean array
				matrix := qr.Bitmap()

				// Print the QR code
				for y := 0; y < len(matrix); y += 2 {
					for x := 0; x < len(matrix[y]); x++ {
						if y+1 < len(matrix) {
							switch {
							case matrix[y][x] && matrix[y+1][x]:
								fmt.Print("█")
							case matrix[y][x] && !matrix[y+1][x]:
								fmt.Print("▀")
							case !matrix[y][x] && matrix[y+1][x]:
								fmt.Print("▄")
							default:
								fmt.Print(" ")
							}
						} else if matrix[y][x] {
							fmt.Print("▀")
						} else {
							fmt.Print(" ")
						}
					}
					fmt.Println()
				}
				fmt.Println("QR code:", evt.Code)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	}

	mux.HandleFunc("/", helloHandler);

	mux.HandleFunc("/qr", generateQr);

	mux.HandleFunc("/connect", connectWithClient);

	handler := cors.Default().Handler(mux)

	error := http.ListenAndServe(":3000", handler);

	log.Fatal(error)
}
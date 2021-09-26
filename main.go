package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"time"

	"github.com/mdp/qrterminal/v3"
)

func main() {
	keep := flag.Bool("keep", false, "if specified, the program does not exit after the first serving of the file, it must hten be terminated manually")
	port := flag.Int("port", 0, "use the given port, otherwise a port will be chosen automatically by the OS")
	big := flag.Bool("big", false, "if specified, the \"pixels\" of the QR code will be bigger")
	flag.Usage = func() {
		fmt.Printf("Usage: %s [OPTIONS] filename\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		return
	}
	filename := flag.Arg(0)
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	finished := make(chan error)
	link, err := serve(ctx, started, finished, *port, *keep, filename)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to start server: %w", err))
		return
	}
	<-started

	fmt.Printf("%s served at %s\n", filename, link)

	printQR(link, *big)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	for {
		select {
		case <-sig:
			cancel()
		case err := <-finished:
			if err != nil {
				fmt.Println(err)
			}
			return
		}
	}
}

func serve(ctx context.Context, started chan struct{}, finished chan error, port int, keep bool, filename string) (string, error) {
	localAddr, err := getOutboundIP()
	if err != nil {
		return "", fmt.Errorf("failed to determine local ip address: %w", err)
	}
	server := &http.Server{
		Addr: fmt.Sprintf("%s:%d", localAddr.String(), port),
	}
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return "", fmt.Errorf("failed to listen: %w", err)
	}
	truePort := listener.Addr().(*net.TCPAddr).Port
	addr := listener.Addr().(*net.TCPAddr).IP.String()

	ctx2, cancel := context.WithCancel(ctx)

	bn := path.Base(filename)
	esc := url.QueryEscape(bn)
	http.HandleFunc("/"+esc, func(rw http.ResponseWriter, r *http.Request) {
		fmt.Printf("Serving to %s \n", r.RemoteAddr)
		rw.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", bn))
		http.ServeFile(rw, r, filename)
		if !keep {
			fmt.Println("Keep not specified -> shutting down...")
			cancel()
		}
	})

	serverFinished := make(chan error)
	go func() {
		<-ctx2.Done()
		fmt.Println("Shutting down...")
		sdCtx, sdCancel := context.WithTimeout(context.Background(), time.Second*10)
		defer sdCancel()
		server.Shutdown(sdCtx)
		err := <-serverFinished
		fmt.Println("Shut down.")
		if err != http.ErrServerClosed {
			finished <- fmt.Errorf("server terminated abnormally: %w", err)
		} else {
			finished <- nil
		}
		close(finished)
	}()
	go func() {
		fmt.Println("Serving...")
		close(started)
		serverFinished <- server.Serve(listener)
		fmt.Println("Server finished.")
	}()
	return fmt.Sprintf("http://%s:%d/%s", addr, truePort, esc), nil
}

// Get preferred outbound ip of this machine
// from https://stackoverflow.com/a/37382208/461202
func getOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, nil
}

func printQR(link string, big bool) {
	config := qrterminal.Config{
		Level:          qrterminal.L,
		BlackChar:      qrterminal.BLACK_BLACK,
		BlackWhiteChar: qrterminal.BLACK_WHITE,
		WhiteChar:      qrterminal.WHITE_WHITE,
		WhiteBlackChar: qrterminal.WHITE_BLACK,
		HalfBlocks:     !big,
		Writer:         os.Stdout,
		QuietZone:      2,
	}
	if big {
		config.BlackChar = qrterminal.BLACK
		config.WhiteChar = qrterminal.WHITE
	}
	qrterminal.GenerateWithConfig(link, config)
}

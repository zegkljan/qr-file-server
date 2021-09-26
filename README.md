# qr-file-server

This simple program serves a single file over HTTP and prints the URL to the console both as text and as QR code, which makes it easy to transfer the file to smartphones.

## Basic usage

    qr-file-server <filename>

will serve the given file using an automatically selected (by the OS) port.
Once the file is first served, the HTTP server shuts down and the program exits.

## Advanced usage

The program accepts a few optional flags which influence its behaviour.
The usage is then

    qr-file-server [-port <port>] [-keep] [-big] <filename>

`-port <port>` will make the program use the specified port instead of a random one

`-keep` will keep the program running after serving the file

`-big` will generate a bigger QR code

## Notes

The program guesstimates the IP address to listen on by dialing an UDP connection to `8.8.8.8` and then reading the local IP address of this connection (regardless whether the connection was successful).
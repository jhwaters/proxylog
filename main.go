package main

import (
	"flag"
	"io"
	"net"
	"os"

	log_ "log"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ReadWriteLogger struct {
	io.ReadWriter
	Log func() *zerolog.Event
	Hex bool
}

func (rwl ReadWriteLogger) Write(b []byte) (int, error) {
	n, err := rwl.ReadWriter.Write(b)
	if rwl.Hex {
		rwl.Log().Hex("data", b).Err(err).Msg("")
	} else {
		rwl.Log().Bytes("data", b).Err(err).Msg("")
	}
	return n, err
}

func Duplex(left io.ReadWriter, right io.ReadWriter) error {
	result := make(chan error)
	go func() {
		_, err := io.Copy(left, right)
		result <- err
	}()
	go func() {
		_, err := io.Copy(right, left)
		result <- err
	}()
	return <-result
}

func AcceptConnections(incoming <-chan net.Conn, remoteAddr net.Addr) (err error) {
	connectionCount := 0
	for client := range incoming {
		server, err := net.Dial(remoteAddr.Network(), remoteAddr.String())
		if err != nil {
			log.Error().
				Str("clientAddr", client.RemoteAddr().String()).
				Err(err).
				Msg("connect to server failed")
			continue
		}
		connectionCount += 1
		if Sync {
			HandleEstablished(client, server, connectionCount)
		} else {
			go HandleEstablished(client, server, connectionCount)
		}
	}
	return
}

func HandleEstablished(client net.Conn, server net.Conn, id int) {
	defer client.Close()
	defer server.Close()
	logger := log.With().
		Int("id", id).
		Str("clientAddr", client.RemoteAddr().String()).
		Str("serverAddr", server.RemoteAddr().String()).
		Logger()
	logger.Log().Msg("connection established")
	var err error
	if NoLog {
		err = Duplex(client, server)
	} else {
		serverLog := log.With().Int("id", id).Str("src", "server").Logger()
		clientLog := log.With().Int("id", id).Str("src", "client").Logger()
		clientLogger := &ReadWriteLogger{client, clientLog.Log, Hex}
		serverLogger := &ReadWriteLogger{server, serverLog.Log, Hex}
		err = Duplex(clientLogger, serverLogger)
	}
	logger.Log().Err(err).Msg("connection closed")
}

func ProxyLog(listenAddr net.Addr, remoteAddr net.Addr) (err error) {
	listener, err := net.Listen(listenAddr.Network(), listenAddr.String())
	if err != nil {
		log.Error().Err(err).Msg("failed to start listener")
		return err
	}
	defer listener.Close()
	log.Info().
		Str("listenAddr", listener.Addr().String()).
		Str("remoteAddr", remoteAddr.String()).
		Msg("listening started")
	connectionChannel := make(chan net.Conn)
	go AcceptConnections(connectionChannel, remoteAddr)
	for {
		client, err := listener.Accept()
		if err != nil {
			log.Error().
				Err(err).
				Msg("accept connection failed")
			continue
		}
		log.Info().
			Str("clientAddr", client.RemoteAddr().String()).
			Msg("connection accepted")
		connectionChannel <- client
	}
}

func Fatal(err interface{}) {
	if err != nil {
		log_.Fatal(err)
	}
}

var Sync bool = false  // used in AcceptConnections
var Hex bool = false   // used in HandleEstablished
var NoLog bool = false // used in HandleEstablished

func main() {
	log_.SetFlags(0)
	log_.SetPrefix("error: ")
	var (
		Append  bool
		Listen  string
		Remote  string
		Color   bool
		Log     string
		Verbose bool
	)

	flag.StringVar(&Listen, "l", "", "listen/local address (required)")
	flag.StringVar(&Remote, "r", "", "remote/server address (required)")
	flag.StringVar(&Log, "o", "", "log to file instead of stderr; - for stdout")
	flag.BoolVar(&Append, "a", false, "append to log file")
	flag.BoolVar(&Sync, "s", false, "force connections to run synchronously")
	flag.BoolVar(&Hex, "x", false, "log bytes in hex format")
	flag.BoolVar(&Color, "c", false, "log with colored console writer")
	flag.BoolVar(&NoLog, "n", false, "do not log data")
	flag.BoolVar(&Verbose, "v", false, "more logging")
	flag.Parse()

	var logFile *os.File
	var err error
	if Listen == "" {
		Fatal("no listen address provided")
	}
	if Remote == "" {
		Fatal("no remote address provided")
	}
	if Log == "" {
		logFile = os.Stderr
	} else if Log == "-" {
		logFile = os.Stdout
	} else {
		if Append {
			logFile, err = os.OpenFile(Log, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		} else {
			logFile, err = os.OpenFile(Log, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
		}
		Fatal(err)
		defer logFile.Close()
	}
	if Verbose {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	}
	if Color {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: logFile})
	} else {
		log.Logger = log.Output(logFile)
	}
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro
	listenAddr, err := net.ResolveTCPAddr("tcp", Listen)
	Fatal(err)
	connectAddr, err := net.ResolveTCPAddr("tcp", Remote)
	Fatal(err)
	err = ProxyLog(listenAddr, connectAddr)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
}

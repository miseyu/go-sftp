package sftp

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"

	"github.com/pkg/sftp"
	gsftp "github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var privateBytes = []byte(`
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEArhp7SqFnXVZAgWREL9Ogs+miy4IU/m0vmdkoK6M97G9NX/Pj
wf8I/3/ynxmcArbt8Rc4JgkjT2uxx/NqR0yN42N1PjO5Czu0dms1PSqcKIJdeUBV
7gdrKSm9Co4d2vwfQp5mg47eG4w63pz7Drk9+VIyi9YiYH4bve7WnGDswn4ycvYZ
slV5kKnjlfCdPig+g5P7yQYud0cDWVwyA0+kxvL6H3Ip+Fu8rLDZn4/P1WlFAIuc
PAf4uEKDGGmC2URowi5eesYR7f6GN/HnBs2776laNlAVXZUmYTUfOGagwLsEkx8x
XdNqntfbs2MOOoK+myJrNtcB9pCrM0H6um19uQIDAQABAoIBABkWr9WdVKvalgkP
TdQmhu3mKRNyd1wCl+1voZ5IM9Ayac/98UAvZDiNU4Uhx52MhtVLJ0gz4Oa8+i16
IkKMAZZW6ro/8dZwkBzQbieWUFJ2Fso2PyvB3etcnGU8/Yhk9IxBDzy+BbuqhYE2
1ebVQtz+v1HvVZzaD11bYYm/Xd7Y28QREVfFen30Q/v3dv7dOteDE/RgDS8Czz7w
jMW32Q8JL5grz7zPkMK39BLXsTcSYcaasT2ParROhGJZDmbgd3l33zKCVc1zcj9B
SA47QljGd09Tys958WWHgtj2o7bp9v1Ufs4LnyKgzrB80WX1ovaSQKvd5THTLchO
kLIhUAECgYEA2doGXy9wMBmTn/hjiVvggR1aKiBwUpnB87Hn5xCMgoECVhFZlT6l
WmZe7R2klbtG1aYlw+y+uzHhoVDAJW9AUSV8qoDUwbRXvBVlp+In5wIqJ+VjfivK
zgIfzomL5NvDz37cvPmzqIeySTowEfbQyq7CUQSoDtE9H97E2wWZhDkCgYEAzJdJ
k+NSFoTkHhfD3L0xCDHpRV3gvaOeew8524fVtVUq53X8m91ng4AX1r74dCUYwwiF
gqTtSSJfx2iH1xKnNq28M9uKg7wOrCKrRqNPnYUO3LehZEC7rwUr26z4iJDHjjoB
uBcS7nw0LJ+0Zeg1IF+aIdZGV3MrAKnrzWPixYECgYBsffX6ZWebrMEmQ89eUtFF
u9ZxcGI/4K8ErC7vlgBD5ffB4TYZ627xzFWuBLs4jmHCeNIJ9tct5rOVYN+wRO1k
/CRPzYUnSqb+1jEgILL6istvvv+DkE+ZtNkeRMXUndWwel94BWsBnUKe0UmrSJ3G
sq23J3iCmJW2T3z+DpXbkQKBgQCK+LUVDNPE0i42NsRnm+fDfkvLP7Kafpr3Umdl
tMY474o+QYn+wg0/aPJIf9463rwMNyyhirBX/k57IIktUdFdtfPicd2MEGETElWv
nN1GzYxD50Rs2f/jKisZhEwqT9YNyV9DkgDdGGdEbJNYqbv0qpwDIg8T9foe8E1p
bdErgQKBgAt290I3L316cdxIQTkJh1DlScN/unFffITwu127WMr28Jt3mq3cZpuM
Aecey/eEKCj+Rlas5NDYKsB18QIuAw+qqWyq0LAKLiAvP1965Rkc4PLScl3MgJtO
QYa37FK0p8NcDeUuF86zXBVutwS5nJLchHhKfd590ks57OROtm29
-----END RSA PRIVATE KEY-----
`)

type SftpHandler interface {
	gsftp.FileReader
	gsftp.FileWriter
	gsftp.FileCmder
	gsftp.FileLister
}

type Server interface {
	ListenAndServe(ctx context.Context)
	Close()
}

type server struct {
	port     int
	user     string
	password string
	handler  SftpHandler
	listener net.Listener
	quit     chan interface{}
	wg       sync.WaitGroup
}

func NewServer(port int, user, password string, handler SftpHandler) Server {
	return &server{
		port:     port,
		user:     user,
		password: password,
		handler:  handler,
		quit:     make(chan interface{}),
	}
}

func (srv *server) Close() {
	if srv.listener != nil {
		close(srv.quit)
		if err := srv.listener.Close(); err != nil {
			log.Printf("close error with %v", err)
		}
		srv.wg.Wait()
	}
}

func (srv *server) ListenAndServe(ctx context.Context) {
	config := &ssh.ServerConfig{
		NoClientAuth:  false,
		ServerVersion: "SSH-2.0-GCS-SFTP",
		AuthLogCallback: func(conn ssh.ConnMetadata, method string, err error) {
			if err != nil {
				log.Printf("Failed %s for user %s from %s ssh2", method, conn.User(), conn.RemoteAddr())
			} else {
				log.Printf("Accepted %s for user %s from %s ssh2", method, conn.User(), conn.RemoteAddr())
			}
		},
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return
	}

	config.PasswordCallback = func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
		log.Printf("user = %s, password = %s", c.User(), string(pass))
		if c.User() == srv.user && string(pass) == srv.password {
			return nil, nil
		}

		return nil, fmt.Errorf("password rejected for %q", c.User())
	}

	config.PublicKeyCallback = func(conn ssh.ConnMetadata, auth ssh.PublicKey) (*ssh.Permissions, error) {
		permissions := &ssh.Permissions{
			CriticalOptions: map[string]string{},
			Extensions:      map[string]string{},
		}

		return permissions, nil
	}

	config.AddHostKey(private)

	// Once a ServerConfig has been configured, connections can be
	// accepted.
	listener, err := net.Listen("tcp", "0.0.0.0:"+strconv.Itoa(srv.port))
	if err != nil {
		return
	}

	srv.listener = listener
	srv.wg.Add(1)

	log.Printf("listening on... address = 0.0.0.0:%s", listener.Addr().String())
Loop:
	for {
		nConn, err := listener.Accept()
		if err != nil {
			select {
			case <-srv.quit:
				break Loop
			default:
				log.Println("accept error", err)
			}
		} else {
			srv.wg.Add(1)
			go func() {
				srv.handleConn(nConn, config)
				srv.wg.Done()
			}()
		}
	}
}

func (srv *server) handleConn(nConn net.Conn, config *ssh.ServerConfig) {
	defer nConn.Close()
	// Before use, a handshake must be performed on the incoming net.Conn.
	sconn, chans, reqs, err := ssh.NewServerConn(nConn, config)
	if err != nil {
		log.Printf("failed to handshake: %s", err)
		return
	}
	log.Printf("login detected: %s", sconn.User())

	// The incoming Request channel must be serviced.
	go ssh.DiscardRequests(reqs)

	// Service the incoming Channel channel.
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			err := newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			log.Printf("unkonw channel type with %v", err)
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("could not accept channel: %s", err)
			return
		}

		go func(in <-chan *ssh.Request) {
			for req := range in {
				ok := false
				switch req.Type {
				case "subsystem":
					if string(req.Payload[4:]) == "sftp" {
						ok = true
					}
				}
				err := req.Reply(ok, nil)
				if err != nil {
					log.Printf("sftp reply error with %v", err)
				}
			}
		}(requests)

		server := sftp.NewRequestServer(channel, sftp.Handlers{
			FileGet:  srv.handler,
			FilePut:  srv.handler,
			FileCmd:  srv.handler,
			FileList: srv.handler,
		})

		if err := server.Serve(); err == io.EOF {
			server.Close()
			log.Printf("sftp client exited session.")
		} else if err != nil {
			log.Printf("sftp server completed with error: %s", err)
		}
	}
}

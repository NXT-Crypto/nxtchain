package gonetic

// GoNetic package to create peer to peer connections

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"nxtchain/nextutils"
	"strings"
	"sync"
	"time"
)

type OutputFunc func(string)

type Peer struct {
	Port           string
	connString     string
	maxPeerList    int
	connectedPeers sync.Map
	listener       net.Listener
	Output         OutputFunc
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

func NewPeer(output OutputFunc, maxPeerList int, port ...string) (*Peer, error) {
	var assignedPort string
	if len(port) > 0 && port[0] != "" {
		assignedPort = port[0]
	} else {
		assignedPort = "0"
	}

	if output == nil {
		output = func(msg string) {
			fmt.Println(msg)
		}
	}

	if maxPeerList < 2 {
		return nil, fmt.Errorf("unable to create a peer with fewer than 2 connections")
	}

	return &Peer{
		Port:           assignedPort,
		Output:         output,
		maxPeerList:    maxPeerList,
		stopChan:       make(chan struct{}),
		connectedPeers: sync.Map{},
	}, nil
}

func (p *Peer) Start() error {
	port := p.Port
	if port == "" {
		port = "0"
	}
	var err error
	p.listener, err = net.Listen("tcp", getLocalIP()+":"+p.Port)
	if err != nil {
		return fmt.Errorf("failed to start node on port %s: %v", p.Port, err)
	}
	addr := p.listener.Addr().(*net.TCPAddr)
	p.connString = fmt.Sprintf("%s:%d", getLocalIP(), addr.Port)
	p.Port = fmt.Sprintf("%d", addr.Port)

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for {
			conn, err := p.listener.Accept()
			if err != nil {
				select {
				case <-p.stopChan:
					return
				default:
					nextutils.Error("Failed to accept connection: %v", err)
					continue
				}
			}

			if len(p.GetConnectedPeers()) >= p.maxPeerList {
				conn.Close()
				continue
			}
			peerID := conn.RemoteAddr().String()
			_, exists := p.connectedPeers.Load(peerID)
			if exists {
				conn.Close()
				continue
			}
			go p.handleConnection(conn)
		}
	}()

	go p.pingPeers()

	<-p.stopChan
	return nil
}

func (p *Peer) SendToPeer(connString string, message string) error {
	if p == nil {
		return fmt.Errorf("peer instance is nil")
	}

	var targetConn net.Conn
	p.connectedPeers.Range(func(key, value interface{}) bool {
		conn := value.(net.Conn)
		if conn.RemoteAddr().String() == connString {
			targetConn = conn
			return false
		}
		return true
	})

	if targetConn == nil {
		return fmt.Errorf("no peer found with connString: %s. Available: %s", connString, p.GetConnectedPeers())
	}

	return p.Send(targetConn, message)
}
func (p *Peer) Send(conn net.Conn, data string) error {
	if conn == nil {
		return fmt.Errorf("connection is nil")
	}
	writer := bufio.NewWriter(conn)
	_, err := writer.WriteString(data + "\n")
	if err != nil {
		return err
	}
	return writer.Flush()
}

func (p *Peer) askForPeers(conn net.Conn) {
	err := p.Send(conn, "GET_PEERS")
	if err != nil {
		log.Printf("Error sending request to peers: %s. Closing connection: %s", err, conn.RemoteAddr().String())
		conn.Close()
		return
	}
}

func (p *Peer) handleConnection(conn net.Conn) {
	defer func() {
		p.connectedPeers.Delete(conn.RemoteAddr().String())
		conn.Close()
	}()
	p.askForPeers(conn)
	reader := bufio.NewReader(conn)
	for {
		messageOrig, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		message := strings.TrimSpace(messageOrig)
		if strings.HasPrefix(message, "GET_") {
			switch message {
			case "GET_PEERS":
				peers := append(p.GetConnectedPeers(), p.GetConnString())
				p.Send(conn, "NEW_PEERS_"+strings.Join(peers, ";"))
			}
		} else if strings.HasPrefix(message, "ERROR_") {
			fmt.Println(strings.Split(message, "_")[1])

		} else if strings.HasPrefix(message, "NEW_PEERS_") {
			peers := strings.Split(message, "_")[2]
			peersArray := strings.Split(peers, ";")
			newPeers := make([]string, 0)
			for _, peer := range peersArray {
				if peer != conn.RemoteAddr().String() && peer != p.GetConnString() {
					_, exists := p.connectedPeers.Load(conn.RemoteAddr().String())
					if !exists {
						newPeers = append(newPeers, peer)
					}
				}
			}
			for _, peer := range newPeers {
				go p.Connect(peer)
			}
		} else {
			if message != "PING" {
				p.Output(messageOrig)
			}
		}
	}
}

func (p *Peer) Stop() {
	close(p.stopChan)
	p.listener.Close()
	p.connectedPeers.Range(func(key, value interface{}) bool {
		conn := value.(net.Conn)
		conn.Close()
		return true
	})
	p.wg.Wait()
}

func (p *Peer) pingPeers() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.connectedPeers.Range(func(key, value interface{}) bool {
				conn := value.(net.Conn)
				err := p.Send(conn, "PING")
				if err != nil {
					nextutils.Error("Failed to ping %s: %v", key, err)
				}
				return true
			})
		case <-p.stopChan:
			return
		}
	}
}

func (p *Peer) Broadcast(message string) {
	p.connectedPeers.Range(func(key, value interface{}) bool {
		conn := value.(net.Conn)
		err := p.Send(conn, message)
		if err != nil {
			nextutils.Error("Failed to broadcast to %s: %v", key, err)
		}
		return true
	})
}

func getLocalIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return ""
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if ip.To4() != nil {
				return ip.String()
			}
		}
	}
	return ""
}

func (p *Peer) Connect(connString string) error {
	conn, err := net.Dial("tcp", connString)
	if err != nil {
		nextutils.Error("failed to connect to peer %s: %v", connString, err)
		return fmt.Errorf("failed to connect to peer %s: %v", connString, err)
	}

	peerID := conn.RemoteAddr().String()
	_, exists := p.connectedPeers.Load(peerID)
	if exists {
		conn.Close()
		nextutils.Error("peer %s is already connected", peerID)
		return fmt.Errorf("peer %s is already connected", peerID)
	}

	p.connectedPeers.Store(peerID, conn)
	go p.handleConnection(conn)
	return nil
}

func (p *Peer) GetConnectedPeers() []string {
	var peers []string

	p.connectedPeers.Range(func(key, value interface{}) bool {
		peerID := key.(string)
		peers = append(peers, peerID)
		return true
	})

	return peers
}

func (p *Peer) GetConnString() string {
	return p.connString
}

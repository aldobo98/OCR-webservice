package sse

import (
	"slices"
	"sync"
)

type SSEHub struct {
	//Read-Write mutex a clients olvasására és írására
	mu sync.RWMutex
	//clients, amely jobID-ként (később userID-ként) tárolja a feliratkozott csatornákat.
	clients map[string][]chan []byte
}

func NewSSeHub() *SSEHub {
	return &SSEHub{
		clients: make(map[string][]chan []byte),
	}
}

func (hub *SSEHub) Publish(topic string, message []byte) {
	//Itt szimplán az adott topichoz tartozó slice-n végig kell menni, és a channelekre rá kell tenni az üzenetet.
	hub.mu.RLock()
	defer hub.mu.Unlock()
	for _, ch := range hub.clients[topic] {
		ch <- message
	}
}

func (hub *SSEHub) Subscribe(topic string) chan []byte {
	//Meg kell nézni, hogy a map adott topic kulcsán létezik-e már slice, és ha igen, akkor létre kell hozni egyet, hozzáadni a slicehez, és visszaküldeni
	sub := make(chan []byte, 10)
	hub.mu.Lock()
	defer hub.mu.Unlock()
	if len(hub.clients[topic]) == 0 {
		hub.clients[topic] = make([]chan []byte, 3)
	}
	hub.clients[topic] = append(hub.clients[topic], sub)
	return sub
}

func (hub *SSEHub) Unsubscribe(topic string, ch chan []byte) {
	//Az adott topic kulcsán lévő slice-ben meg kell keresni a channelt, és ha benne van, törölni kell.
	hub.mu.Lock()
	defer hub.mu.Unlock()
	if len(hub.clients[topic]) == 0 {
		return
	}
	idx := slices.Index(hub.clients[topic], ch)
	if idx == -1 {
		return
	}
	hub.clients[topic] = append(hub.clients[topic][:idx], hub.clients[topic][idx+1:]...)
}

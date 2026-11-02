package broker

type MessageBroker interface {
	Publish(topic string, message []byte)
	Subscribe(topic string) chan []byte
	Unsubscribe(topic string, ch chan []byte)
}

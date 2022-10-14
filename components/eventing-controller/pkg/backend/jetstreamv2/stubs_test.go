package jetstreamv2 //nolint:nolintlint,testpackage

import (
	"github.com/nats-io/nats.go"
)

type jetStreamContextStub struct {
	consumerInfoError error
	consumerInfo      *nats.ConsumerInfo

	addConsumerError error
	addConsumer      *nats.ConsumerInfo

	subscribe      *nats.Subscription
	subscribeError error

	update      *nats.ConsumerInfo
	updateError error
}

func (j jetStreamContextStub) Streams(_ ...nats.JSOpt) <-chan *nats.StreamInfo {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) Consumers(_ string, _ ...nats.JSOpt) <-chan *nats.ConsumerInfo {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) ObjectStoreNames(_ ...nats.ObjectOpt) <-chan string {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) ObjectStores(_ ...nats.ObjectOpt) <-chan nats.ObjectStore {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) Publish(_ string, _ []byte, _ ...nats.PubOpt) (*nats.PubAck, error) {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) PublishMsg(_ *nats.Msg, _ ...nats.PubOpt) (*nats.PubAck, error) {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) PublishAsync(_ string, _ []byte, _ ...nats.PubOpt) (nats.PubAckFuture, error) {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) PublishMsgAsync(_ *nats.Msg, _ ...nats.PubOpt) (nats.PubAckFuture, error) {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) PublishAsyncPending() int {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) PublishAsyncComplete() <-chan struct{} {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) Subscribe(_ string, _ nats.MsgHandler, _ ...nats.SubOpt) (*nats.Subscription, error) {
	return j.subscribe, j.subscribeError
}

func (j jetStreamContextStub) SubscribeSync(_ string, _ ...nats.SubOpt) (*nats.Subscription, error) {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) ChanSubscribe(_ string, _ chan *nats.Msg, _ ...nats.SubOpt) (*nats.Subscription, error) {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) ChanQueueSubscribe(_, _ string,
	_ chan *nats.Msg, _ ...nats.SubOpt) (*nats.Subscription, error) {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) QueueSubscribe(_, _ string, _ nats.MsgHandler,
	_ ...nats.SubOpt) (*nats.Subscription, error) {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) QueueSubscribeSync(_, _ string, _ ...nats.SubOpt) (*nats.Subscription, error) {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) PullSubscribe(_, _ string, _ ...nats.SubOpt) (*nats.Subscription, error) {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) AddStream(_ *nats.StreamConfig, _ ...nats.JSOpt) (*nats.StreamInfo, error) {
	panic("really implement me")
}

func (j jetStreamContextStub) UpdateStream(_ *nats.StreamConfig, _ ...nats.JSOpt) (*nats.StreamInfo, error) {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) DeleteStream(_ string, _ ...nats.JSOpt) error {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) StreamInfo(_ string, _ ...nats.JSOpt) (*nats.StreamInfo, error) {
	panic("really implement me")
}

func (j jetStreamContextStub) PurgeStream(_ string, _ ...nats.JSOpt) error {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) StreamsInfo(_ ...nats.JSOpt) <-chan *nats.StreamInfo {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) StreamNames(_ ...nats.JSOpt) <-chan string {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) GetMsg(_ string, _ uint64, _ ...nats.JSOpt) (*nats.RawStreamMsg, error) {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) GetLastMsg(_, _ string, _ ...nats.JSOpt) (*nats.RawStreamMsg, error) {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) DeleteMsg(_ string, _ uint64, _ ...nats.JSOpt) error {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) SecureDeleteMsg(_ string, _ uint64, _ ...nats.JSOpt) error {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) AddConsumer(_ string, _ *nats.ConsumerConfig,
	_ ...nats.JSOpt) (*nats.ConsumerInfo, error) {
	return j.addConsumer, j.addConsumerError
}

func (j jetStreamContextStub) UpdateConsumer(_ string, _ *nats.ConsumerConfig,
	_ ...nats.JSOpt) (*nats.ConsumerInfo, error) {
	return j.update, j.updateError
}

func (j jetStreamContextStub) DeleteConsumer(_, _ string, _ ...nats.JSOpt) error {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) ConsumerInfo(_, _ string, _ ...nats.JSOpt) (*nats.ConsumerInfo, error) {
	return j.consumerInfo, j.consumerInfoError
}

func (j jetStreamContextStub) ConsumersInfo(_ string, _ ...nats.JSOpt) <-chan *nats.ConsumerInfo {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) ConsumerNames(_ string, _ ...nats.JSOpt) <-chan string {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) AccountInfo(_ ...nats.JSOpt) (*nats.AccountInfo, error) {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) KeyValue(_ string) (nats.KeyValue, error) {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) CreateKeyValue(_ *nats.KeyValueConfig) (nats.KeyValue, error) {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) DeleteKeyValue(_ string) error {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) ObjectStore(_ string) (nats.ObjectStore, error) {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) CreateObjectStore(_ *nats.ObjectStoreConfig) (nats.ObjectStore, error) {
	// TODO implement me
	panic("implement me")
}

func (j jetStreamContextStub) DeleteObjectStore(_ string) error {
	// TODO implement me
	panic("implement me")
}

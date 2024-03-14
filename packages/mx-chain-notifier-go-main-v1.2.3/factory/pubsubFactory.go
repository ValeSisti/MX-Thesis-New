package factory

import (
	"github.com/multiversx/mx-chain-core-go/marshal"
	"github.com/multiversx/mx-chain-notifier-go/common"
	"github.com/multiversx/mx-chain-notifier-go/config"
	"github.com/multiversx/mx-chain-notifier-go/dispatcher"
	"github.com/multiversx/mx-chain-notifier-go/process"
	"github.com/multiversx/mx-chain-notifier-go/rabbitmq"
)

// CreatePublisher creates publisher component
func CreatePublisher(
	apiType string,
	config config.MainConfig,
	marshaller marshal.Marshalizer,
	commonHub dispatcher.Hub,
) (process.Publisher, error) {
	//! -------------------- NEW CODE --------------------
	log.Debug("*** CreatePublisher called***")
	//! ---------------- END OF NEW CODE -----------------	
	switch apiType {
	case common.MessageQueuePublisherType:
		//! -------------------- NEW CODE --------------------
		log.Debug("*** case common.MessageQueuePublisherType ***")
		//! ---------------- END OF NEW CODE -----------------		
		return createRabbitMqPublisher(config.RabbitMQ, marshaller)
	case common.WSPublisherType:
		//! -------------------- NEW CODE --------------------
		log.Debug("*** case common.MessageQueuePublisherType ***")
		//! ---------------- END OF NEW CODE -----------------				
		return createWSPublisher(commonHub)
	default:
		return nil, common.ErrInvalidAPIType
	}
}

func createRabbitMqPublisher(config config.RabbitMQConfig, marshaller marshal.Marshalizer) (rabbitmq.PublisherService, error) {
	//! -------------------- NEW CODE --------------------
	log.Debug("***createRabbitMqPublisher called***")
	//! ---------------- END OF NEW CODE -----------------		
	rabbitClient, err := rabbitmq.NewRabbitMQClient(config.Url)
	if err != nil {
		return nil, err
	}

	rabbitMqPublisherArgs := rabbitmq.ArgsRabbitMqPublisher{
		Client:     rabbitClient,
		Config:     config,
		Marshaller: marshaller,
	}
	rabbitPublisher, err := rabbitmq.NewRabbitMqPublisher(rabbitMqPublisherArgs)
	if err != nil {
		//! -------------------- NEW CODE --------------------
		log.Debug("***Error: during NewRabbitMqPublisher***")
		//! ---------------- END OF NEW CODE -----------------		
		return nil, err
	}

	return process.NewPublisher(rabbitPublisher)
}

func createWSPublisher(commonHub dispatcher.Hub) (process.Publisher, error) {
	return process.NewPublisher(commonHub)
}

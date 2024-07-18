package conn

import (
	sls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/aliyun/aliyun-log-go-sdk/producer"
)

var producerInstance *producer.Producer
var logHubClient sls.ClientInterface

func NewLogHub(endpoint string) {
	if ENV == "local" || ENV == "local-docker" {
		endpoint = endpoint + ".aliyuncs.com"
	} else {
		endpoint = endpoint + "-internal.aliyuncs.com"
	}

	credit := sls.NewStaticCredentialsProvider(AliyunID, AliyunSecret, "")
	logHubClient = sls.CreateNormalInterfaceV2(endpoint, credit)

	producerConfig := producer.GetDefaultProducerConfig()
	producerConfig.Endpoint = endpoint
	producerConfig.CredentialsProvider = credit
	producerInstance = producer.InitProducer(producerConfig)
	// 是为了防止丢数据的
	//ch := make(chan os.Signal)
	//signal.Notify(ch, os.Kill, os.Interrupt)
	producerInstance.Start()
}

func GetLogHub() sls.ClientInterface {
	return logHubClient
}

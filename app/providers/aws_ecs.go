package providers

import (
	"context"
	"fmt"
	"log"

	"github.com/acouvreur/sablier/app/instance"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

type AWSElasticContainerServiceProvider struct {
	Client          *ecs.Client
	desiredReplicas int
}

func NewAWSElasticContainerServiceProvider() (*AWSElasticContainerServiceProvider, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	client := ecs.NewFromConfig(cfg)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	return &AWSElasticContainerServiceProvider{
		Client:          client,
		desiredReplicas: 1,
	}, nil
}

func (provider *AWSElasticContainerServiceProvider) Start(name string) (instance.State, error) {
	return provider.scale(name, int32(provider.desiredReplicas))
}

func (provider *AWSElasticContainerServiceProvider) Stop(name string) (instance.State, error) {
	return provider.scale(name, int32(0))
}

func (provider *AWSElasticContainerServiceProvider) scale(name string, replicas int32) (instance.State, error) {
	ctx := context.Background()

	params := ecs.UpdateServiceInput{
		Service:      &name,
		DesiredCount: &replicas,
	}
	_, err := provider.Client.UpdateService(ctx, &params)

	if err != nil {
		return instance.ErrorInstanceState(name, err, provider.desiredReplicas)
	}

	return instance.State{
		Name:            name,
		CurrentReplicas: 0,
		DesiredReplicas: provider.desiredReplicas,
		Status:          instance.NotReady,
	}, err
}

func (provider *AWSElasticContainerServiceProvider) GetState(name string) (instance.State, error) {
	ctx := context.Background()

	params := ecs.DescribeServicesInput{
		Services: []string{name},
	}
	spec, err := provider.Client.DescribeServices(ctx, &params)

	if err != nil {
		return instance.ErrorInstanceState(name, err, provider.desiredReplicas)
	}

	if len(spec.Services) != 1 {
		return instance.ErrorInstanceState(name, fmt.Errorf("not exactly 1 service found for %s (found %d: %v", name, len(spec.Services), spec.Services), provider.desiredReplicas)
	}

	service := spec.Services[0]

	// The status of the service. The valid values are ACTIVE, DRAINING, or INACTIVE.
	switch *service.Status {
	case "DRAINING", "INACTIVE":
		return instance.NotReadyInstanceState(name, 0, provider.desiredReplicas)
	case "ACTIVE":
		if service.RunningCount < int32(provider.desiredReplicas) {
			return instance.NotReadyInstanceState(name, int(service.RunningCount), provider.desiredReplicas)
		}
		return instance.ReadyInstanceState(name, provider.desiredReplicas)

	default:
		return instance.UnrecoverableInstanceState(name, fmt.Sprintf("ecs service status \"%s\" not handled", *service.Status), provider.desiredReplicas)
	}
}

func (provider *AWSElasticContainerServiceProvider) NotifyInstanceStopped(ctx context.Context, instance chan<- string) {
	// Unsupported
}

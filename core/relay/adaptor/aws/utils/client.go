package utils

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/smithy-go/auth/bearer"
	"github.com/labring/aiproxy/core/relay/meta"
)

type AwsConfig struct {
	Region string
	AK     string
	SK     string
	APIKey string
}

func GetAwsConfigFromKey(key string) (*AwsConfig, error) {
	split := strings.Split(key, "|")
	if len(split) == 2 {
		return &AwsConfig{
			Region: split[0],
			APIKey: split[1],
		}, nil
	}

	if len(split) != 3 {
		return nil, errors.New("invalid key format")
	}

	return &AwsConfig{
		Region: split[0],
		AK:     split[1],
		SK:     split[2],
	}, nil
}

func AwsClient(config *AwsConfig) *bedrockruntime.Client {
	options := bedrockruntime.Options{
		Region: config.Region,
	}
	if config.APIKey != "" {
		options.BearerAuthTokenProvider = bearer.TokenProviderFunc(
			func(ctx context.Context) (bearer.Token, error) {
				return bearer.Token{Value: config.APIKey}, nil
			},
		)
		options.AuthSchemePreference = []string{"httpBearerAuth"}
	} else {
		options.Credentials = aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(config.AK, config.SK, ""),
		)
	}

	return bedrockruntime.New(options)
}

func awsClientFromKey(key string) (*bedrockruntime.Client, error) {
	config, err := GetAwsConfigFromKey(key)
	if err != nil {
		return nil, err
	}

	return AwsClient(config), nil
}

const AwsClientKey = "aws_client"

func AwsClientFromMeta(meta *meta.Meta) (*bedrockruntime.Client, error) {
	awsClientI, ok := meta.Get(AwsClientKey)
	if ok {
		v, ok := awsClientI.(*bedrockruntime.Client)
		if !ok {
			panic(fmt.Sprintf("aws client type error: %T, %v", v, v))
		}

		return v, nil
	}

	awsClient, err := awsClientFromKey(meta.Channel.Key)
	if err != nil {
		return nil, err
	}

	meta.Set(AwsClientKey, awsClient)

	return awsClient, nil
}

func AwsRegionFromMeta(meta *meta.Meta) (string, error) {
	config, err := GetAwsConfigFromKey(meta.Channel.Key)
	if err != nil {
		return "", err
	}

	return config.Region, nil
}

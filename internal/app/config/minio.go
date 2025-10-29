package config

import (
	"context"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var MinioClient *minio.Client

func InitMinio() {
	endpoint := "localhost:9000" // Nginx балансирует запросы на кластер MinIO
	accessKeyID := "minio"
	secretAccessKey := "minio124"
	useSSL := false

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к MinIO: %v", err)
	}

	MinioClient = client
	log.Println("✅ Подключение к MinIO успешно")

	// Проверим, что bucket test существует
	ctx := context.Background()
	exists, err := MinioClient.BucketExists(ctx, "test")
	if err != nil {
		log.Fatalf("Ошибка проверки bucket: %v", err)
	}
	if !exists {
		err = MinioClient.MakeBucket(ctx, "test", minio.MakeBucketOptions{})
		if err != nil {
			log.Fatalf("Ошибка создания bucket: %v", err)
		}
		log.Println("🪣 Bucket 'test' создан")
	} else {
		log.Println("🪣 Bucket 'test' найден")
	}
}

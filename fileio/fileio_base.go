// fileio ファイルIOシステムパッケージ
package fileio // パッケージ名はディレクトリ名と同じにする

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/goccy/go-json"
)

// ---- Global Variable

// ---- Package Global Variable

//---- public function ----

//---- ファイル読み書き ----

// FileIoRead (public)ファイルを一括で読み込む
func FileIoRead(filename string) ([]byte, error) {

	file, errOpen := os.Open(filename) // ファイルを開く
	if errOpen != nil {
		return nil, errOpen
	}
	defer file.Close()

	contents, errRead := io.ReadAll(file) // ファイル全体をメモリへ読み込む
	if errRead != nil {
		return nil, errRead
	}

	return contents, nil
}

// FileIoWrite (public)ファイルを一括で書き込む
func FileIoWrite(filename string, fileContents []byte, isAppend bool) error {

	flag := os.O_WRONLY | os.O_CREATE
	if isAppend == true {
		flag = os.O_RDWR | os.O_CREATE | os.O_APPEND
	}
	file, errOpen := os.OpenFile(filename, flag, 0666) // ファイルを開く
	if errOpen != nil {
		return errOpen
	}
	defer file.Close()

	_, errWrite := file.Write(fileContents)
	if errWrite != nil {
		return errWrite
	}

	return nil
}

//---- CSVファイル読み書き ----

// FileIoCsvRead (public)Csvファイルを一括で読み込む
func FileIoCsvRead(filename string) ([][]string, error) {

	file, errOpen := os.Open(filename) // ファイルを開く
	if errOpen != nil {
		return nil, errOpen
	}
	defer file.Close()

	readCsv := csv.NewReader(file)
	csvContents, errRead := readCsv.ReadAll() // csvを一度に全て読み込む
	if errRead != nil {
		return nil, errRead
	}

	return csvContents, nil
}

// FileIoCsvWrite (public)Csvファイルを一括で書き込む
func FileIoCsvWrite(filename string, csvContents [][]string, isAppend bool) error {

	flag := os.O_WRONLY | os.O_CREATE
	if isAppend == true {
		flag = os.O_RDWR | os.O_CREATE | os.O_APPEND
	}
	file, errOpen := os.OpenFile(filename, flag, 0666) // ファイルを開く
	if errOpen != nil {
		return errOpen
	}
	defer file.Close()

	writeCsv := csv.NewWriter(file)
	errWrite := writeCsv.WriteAll(csvContents) // csvを一度に全て書き込む
	if errWrite != nil {
		return errWrite
	}

	return nil
}

//---- jsonファイル(UTF-8 BOMなし)読み書き ----

// FileIoJsonRead (public)Jsonファイルを一括で一つの構造体に読み込む
func FileIoJsonRead(filename string, body any) error {

	cont, errRead := FileIoRead(filename)
	if errRead != nil {
		return errRead
	}

	json.Unmarshal(cont, body)
	return nil
}

// FileIoJsonWrite (public)Jsonファイルを一括で書き込む
func FileIoJsonWrite(filename string, body any, isAppend bool) error {

	jsonContents, _ := json.Marshal(body)
	errWrite := FileIoWrite(filename, jsonContents, isAppend)
	if errWrite != nil {
		return errWrite
	}

	return nil
}

// S3へのファイルアップロード(C:\Users\<YourUsername>\.aws\credentialsを使用)
func UploadFileToS3(bucketName, filePath, key string) error {
	// Load AWS config
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Upload the file to S3
	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	slog.Info("File uploaded successfully", "bucket", bucketName, "key", key)
	return nil
}

//---- private function ----

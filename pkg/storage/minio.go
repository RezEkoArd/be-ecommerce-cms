// Package storage membungkus MinIO/S3 untuk penyimpanan gambar produk.
//
// Skema yang dipakai:
//   - Upload  → presigned PUT URL, browser kirim file langsung ke MinIO.
//     Kredensial tidak pernah keluar dari server.
//   - Baca    → URL publik permanen (bucket ber-policy read-only anonim),
//     supaya bisa di-cache browser/CDN.
package storage

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// PresignExpiry = masa berlaku URL upload. Cukup untuk satu sesi unggah,
// tidak terlalu lama agar URL bocor tidak bisa dipakai berhari-hari.
const PresignExpiry = 15 * time.Minute

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	PublicURL string
}

type Storage struct {
	client *minio.Client
	cfg    Config
}

func New(cfg Config) (*Storage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("storage.New: %w", err)
	}
	return &Storage{client: client, cfg: cfg}, nil
}

// PresignedUpload membuat URL PUT bertanda tangan untuk satu objek.
// objectKey dikembalikan agar pemanggil menyimpannya ke DB.
func (s *Storage) PresignedUpload(ctx context.Context, prefix, filename string) (uploadURL, objectKey string, err error) {
	// Nama file dari user tidak dipakai apa adanya — cegah tabrakan
	// dan karakter aneh dengan UUID, ekstensi tetap dipertahankan.
	ext := path.Ext(filename)
	objectKey = path.Join(prefix, uuid.NewString()+ext)

	u, err := s.client.PresignedPutObject(ctx, s.cfg.Bucket, objectKey, PresignExpiry)
	if err != nil {
		return "", "", fmt.Errorf("storage.PresignedUpload: %w", err)
	}
	return u.String(), objectKey, nil
}

// PublicURL membangun URL permanen untuk sebuah objek.
func (s *Storage) PublicURL(objectKey string) string {
	base := s.cfg.PublicURL
	if base == "" {
		scheme := "http"
		if s.cfg.UseSSL {
			scheme = "https"
		}
		base = scheme + "://" + s.cfg.Endpoint
	}
	return fmt.Sprintf("%s/%s/%s", base, s.cfg.Bucket, objectKey)
}

// ObjectKeyFromURL mengembalikan object key dari URL publik.
// Dipakai saat menghapus gambar yang tersimpan sebagai URL penuh di DB.
func (s *Storage) ObjectKeyFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	// Path berbentuk /<bucket>/<key...>
	prefix := "/" + s.cfg.Bucket + "/"
	if len(u.Path) > len(prefix) && u.Path[:len(prefix)] == prefix {
		return u.Path[len(prefix):]
	}
	return ""
}

func (s *Storage) Remove(ctx context.Context, objectKey string) error {
	err := s.client.RemoveObject(ctx, s.cfg.Bucket, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("storage.Remove: %w", err)
	}
	return nil
}

// BucketExists dipakai saat startup untuk memastikan konfigurasi benar.
func (s *Storage) BucketExists(ctx context.Context) (bool, error) {
	ok, err := s.client.BucketExists(ctx, s.cfg.Bucket)
	if err != nil {
		return false, fmt.Errorf("storage.BucketExists: %w", err)
	}
	return ok, nil
}

// mini-GED — a tiny document-management demo backed by Garage (S3-compatible).
//
// Usage:
//   S3_ENDPOINT=http://<lxc-ip>:3900 \
//   S3_REGION=garage-demo \
//   S3_BUCKET=demo-documents \
//   S3_ACCESS_KEY=xxxx \
//   S3_SECRET_KEY=xxxx \
//   go run main.go
//
// Then browse to http://localhost:8080
package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	s3Client *s3.Client
	bucket   string
	basePath string // e.g. "/has-app" when served behind a path-prefixing reverse proxy
)

type Doc struct {
	Key          string
	Size         int64
	LastModified time.Time
}

func main() {
	endpoint := mustEnv("S3_ENDPOINT")
	region := envOr("S3_REGION", "garage-demo")
	bucket = mustEnv("S3_BUCKET")
	accessKey := mustEnv("S3_ACCESS_KEY")
	secretKey := mustEnv("S3_SECRET_KEY")
	basePath = envOr("BASE_PATH", "")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}

	s3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true // required for Garage (no wildcard DNS in this demo)
	})

	mux := http.NewServeMux()
	mux.HandleFunc(basePath+"/", handleIndex)
	mux.HandleFunc(basePath+"/upload", handleUpload)
	mux.HandleFunc(basePath+"/download", handleDownload)
	mux.HandleFunc(basePath+"/delete", handleDelete)
	mux.HandleFunc(basePath+"/overwrite-demo", handleOverwriteDemo)

	addr := envOr("LISTEN_ADDR", ":8080")
	log.Printf("mini-GED listening on %s (bucket=%s endpoint=%s)", addr, bucket, endpoint)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// ---------- handlers ----------

func handleIndex(w http.ResponseWriter, r *http.Request) {
	docs, err := listDocs(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := pageTmpl.Execute(w, struct {
		Bucket   string
		BasePath string
		Docs     []Doc
		Flash    string
	}{
		Bucket:   bucket,
		BasePath: basePath,
		Docs:     docs,
		Flash:    r.URL.Query().Get("flash"),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	_, err = s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(header.Filename),
		Body:   file,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectFlash(w, r, fmt.Sprintf("Document '%s' déposé avec succès.", header.Filename))
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}
	out, err := s3Client.GetObject(r.Context(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer out.Body.Close()

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, key))
	io.Copy(w, out.Body)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}
	_, err := s3Client.DeleteObject(r.Context(), &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectFlash(w, r, fmt.Sprintf("Document '%s' supprimé.", key))
}

// handleOverwriteDemo is the "punchline" feature for the presentation:
// it writes two different versions of the SAME key back-to-back, then
// shows that only the latest one survives — no version history kept.
// This is a live, visual illustration of the versioning gap discussed
// in the slides.
func handleOverwriteDemo(w http.ResponseWriter, r *http.Request) {
	key := "demo-versioning-test.txt"
	ctx := r.Context()

	v1 := fmt.Sprintf("Version 1 — écrite à %s", time.Now().Format(time.RFC3339))
	if _, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket), Key: aws.String(key), Body: strReader(v1),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	time.Sleep(500 * time.Millisecond)

	v2 := fmt.Sprintf("Version 2 — écrite à %s (écrase la version 1)", time.Now().Format(time.RFC3339))
	if _, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket), Key: aws.String(key), Body: strReader(v2),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	redirectFlash(w, r, fmt.Sprintf(
		"Démo écrasement : deux écritures envoyées sur '%s'. Téléchargez le fichier — seule la version 2 existe, la version 1 a été perdue (pas de versioning natif dans Garage).",
		key))
}

// ---------- helpers ----------

func listDocs(ctx context.Context) ([]Doc, error) {
	out, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return nil, err
	}
	docs := make([]Doc, 0, len(out.Contents))
	for _, obj := range out.Contents {
		docs = append(docs, Doc{
			Key:          aws.ToString(obj.Key),
			Size:         aws.ToInt64(obj.Size),
			LastModified: aws.ToTime(obj.LastModified),
		})
	}
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].LastModified.After(docs[j].LastModified)
	})
	return docs, nil
}

func redirectFlash(w http.ResponseWriter, r *http.Request, msg string) {
	http.Redirect(w, r, basePath+"/?flash="+template.URLQueryEscaper(msg), http.StatusSeeOther)
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("missing required env var %s", k)
	}
	return v
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func strReader(s string) *stringReadSeeker {
	return &stringReadSeeker{s: s}
}

// minimal io.ReadSeeker over a string, avoids pulling in strings.NewReader
// friction with s3 manager needing seekability for content-length calc
type stringReadSeeker struct {
	s   string
	pos int64
}

func (r *stringReadSeeker) Read(p []byte) (int, error) {
	if r.pos >= int64(len(r.s)) {
		return 0, io.EOF
	}
	n := copy(p, r.s[r.pos:])
	r.pos += int64(n)
	return n, nil
}

func (r *stringReadSeeker) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = r.pos + offset
	case io.SeekEnd:
		newPos = int64(len(r.s)) + offset
	}
	r.pos = newPos
	return newPos, nil
}

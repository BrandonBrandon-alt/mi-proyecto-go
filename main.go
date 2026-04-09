package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ImageInfo struct {
	Base64Data string
	MimeType   string
}

type PageData struct {
	Images []ImageInfo
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Servidor de Imágenes Aleatorias</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css" rel="stylesheet">
    <style>
        body { background-color: #f8f9fa; }
        .image-card {
            transition: transform 0.2s;
            border-radius: 10px;
            overflow: hidden;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
            background-color: #fff;
            height: 100%;
        }
        .image-card:hover {
            transform: scale(1.03);
            box-shadow: 0 10px 15px rgba(0,0,0,0.15);
        }
        .img-custom { width: 100%; height: 250px; object-fit: cover; }
    </style>
</head>
<body>
    <div class="container py-5">
        <div class="row mb-5 text-center">
            <div class="col-12">
                <h1 class="display-4 fw-bold text-primary">Galería de Imágenes</h1>
                <p class="lead text-muted">Mostrando una selección aleatoria de imágenes codificadas en Base64.</p>
                <div class="badge bg-secondary mb-3">Imágenes cargadas: {{len .Images}}</div>
                <div>
                    <button onclick="window.location.reload()" class="btn btn-primary">Cargar Nuevas Imágenes</button>
                </div>
            </div>
        </div>
        <div class="row g-4 d-flex align-items-stretch">
            {{if eq (len .Images) 0}}
                <div class="col-12 text-center">
                    <div class="alert alert-warning" role="alert">
                        No se encontraron imágenes válidas (.png, .jpg, .jpeg) en la carpeta "images".
                    </div>
                </div>
            {{else}}
                {{range .Images}}
                <div class="col-12 col-sm-6 col-md-4 col-lg-3">
                    <div class="image-card">
                        <img src="data:{{.MimeType}};base64,{{.Base64Data}}" class="img-custom" alt="Imagen Aleatoria">
                    </div>
                </div>
                {{end}}
            {{end}}
        </div>
    </div>
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/js/bootstrap.bundle.min.js"></script>
</body>
</html>`

func isValidExtension(ext string) bool {
	ext = strings.ToLower(ext)
	return ext == ".png" || ext == ".jpg" || ext == ".jpeg"
}

func getMimeType(ext string) string {
	if strings.ToLower(ext) == ".png" {
		return "image/png"
	}
	return "image/jpeg"
}

func handler(w http.ResponseWriter, r *http.Request) {
	imagesDir := "images"
	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		log.Printf("Error leyendo directorio de imágenes: %v", err)
		entries = []os.DirEntry{}
	}

	var validFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if isValidExtension(ext) {
			validFiles = append(validFiles, filepath.Join(imagesDir, entry.Name()))
		}
	}

	var selectedImageInfos []ImageInfo
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	if len(validFiles) > 0 {
		rng.Shuffle(len(validFiles), func(i, j int) {
			validFiles[i], validFiles[j] = validFiles[j], validFiles[i]
		})
		numImages := rng.Intn(len(validFiles)) + 1
		for _, file := range validFiles[:numImages] {
			data, err := os.ReadFile(file)
			if err != nil {
				log.Printf("Error leyendo archivo %s: %v", file, err)
				continue
			}
			selectedImageInfos = append(selectedImageInfos, ImageInfo{
				Base64Data: base64.StdEncoding.EncodeToString(data),
				MimeType:   getMimeType(filepath.Ext(file)),
			})
		}
	}

	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, PageData{Images: selectedImageInfos}); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

func main() {
	// CORRECCIÓN: puerto recibido como argumento -port
	// Esto permite que systemd lo inicie con: ./srvimg -port 8081
	port := flag.String("port", "8080", "Puerto donde escucha el servidor de imágenes")
	flag.Parse()

	http.HandleFunc("/", handler)
	log.Printf("Servidor de imágenes escuchando en http://localhost:%s", *port)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatalf("Error iniciando servidor: %v", err)
	}
}

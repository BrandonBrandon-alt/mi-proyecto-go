// Package main es el punto de entrada del servidor de imágenes aleatorias.
// Al ejecutar "go run main.go", Go busca este paquete y llama a la función main().
package main

import (
	"bytes"
	"encoding/base64"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ImageInfo representa una imagen lista para ser incrustada en HTML.
// Almacena el contenido de la imagen codificado en Base64 y su tipo MIME.
type ImageInfo struct {
	Base64Data string // Contenido binario de la imagen convertido a texto Base64.
	MimeType   string // Tipo MIME de la imagen, p.ej. "image/png" o "image/jpeg".
}

// PageData contiene los datos que se inyectan en la plantilla HTML al renderizar la página.
type PageData struct {
	Images []ImageInfo // Lista de imágenes seleccionadas aleatoriamente para mostrar.
}

// htmlTemplate es la plantilla HTML completa de la página web.
// Usa la sintaxis de Go templates ({{...}}) para insertar datos dinámicos en tiempo de ejecución.
// La cadena multilínea (delimitada por backticks) incluye CSS embebido y bloques de Bootstrap 5.3.
const htmlTemplate = `<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Servidor de Imágenes Aleatorias</title>
    <!-- Bootstrap 5.3 CSS -->
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-QWTKZyjpPEjISv5WaRU9OFeRpok6YctnYmDr5pNlyT2bRjXh0JMhjY6hW+ALEwIH" crossorigin="anonymous">
    <style>
        body {
            background-color: #f8f9fa;
        }
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
        .img-custom {
            width: 100%;
            height: 250px;
            object-fit: cover;
        }
    </style>
</head>
<body>
    <div class="container py-5">
        <div class="row mb-5 text-center">
            <div class="col-12">
                <h1 class="display-4 fw-bold text-primary">Galería de Imágenes</h1>
                <p class="lead text-muted">Mostrando una selección aleatoria de imágenes codificadas en Base64.</p>
                {{/* len .Images devuelve el número de imágenes en la lista */}}
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
                {{/* range itera sobre cada ImageInfo en la lista, exponiendo sus campos con {{.Campo}} */}}
                {{range .Images}}
                <div class="col-12 col-sm-6 col-md-4 col-lg-3">
                    <div class="image-card">
                        {{/* La imagen se incrusta directamente usando el esquema data URI: data:[MIME];base64,[datos] */}}
                        <img src="data:{{.MimeType}};base64,{{.Base64Data}}" class="img-custom" alt="Imagen Aleatoria">
                    </div>
                </div>
                {{end}}
            {{end}}
        </div>
    </div>

    <!-- Bootstrap 5.3 JS Bundle -->
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/js/bootstrap.bundle.min.js" integrity="sha384-YvpcrYf0tY3lHB60NNkmxc5s9fDVZLESaAA55NDzOxhy9GkcIdslK1eN7N6jIeHz" crossorigin="anonymous"></script>
</body>
</html>`

// isValidExtension informa si la extensión dada corresponde a un formato de imagen soportado.
// La comparación se hace en minúsculas para aceptar también extensiones como .PNG o .JPG.
func isValidExtension(ext string) bool {
	ext = strings.ToLower(ext)
	return ext == ".png" || ext == ".jpg" || ext == ".jpeg"
}

// getMimeType devuelve el tipo MIME correspondiente a la extensión de archivo recibida.
// Es necesario para construir correctamente el atributo src de las imágenes en Base64.
func getMimeType(ext string) string {
	ext = strings.ToLower(ext)
	if ext == ".png" {
		return "image/png"
	}
	return "image/jpeg"
}

// handler atiende cada petición HTTP entrante.
// Lee las imágenes del directorio "images", codifica una selección aleatoria en Base64,
// renderiza la plantilla HTML con esos datos y envía la respuesta al cliente.
func handler(w http.ResponseWriter, r *http.Request) {
	imagesDir := "images"

	// Read files from images directory
	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("El directorio %s no existe.", imagesDir)
		} else {
			log.Printf("Error leyendo el directorio de imágenes: %v", err)
		}
		// Vamos a seguir, mostrará que no hay imágenes en lugar de un error feo
		entries = []os.DirEntry{}
	}

	// Recorrer las entradas del directorio y conservar solo archivos con extensión válida.
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
	// El generador usa el tiempo actual como semilla para garantizar aleatoriedad en cada petición.
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// If there are valid files, pick a random number of them (between 1 and len(validFiles))
	if len(validFiles) > 0 {
		// Shuffle valid files to ensure uniqueness
		rng.Shuffle(len(validFiles), func(i, j int) {
			validFiles[i], validFiles[j] = validFiles[j], validFiles[i]
		})

		// Random number of images to display (at least 1, at most total available)
		numImages := rng.Intn(len(validFiles)) + 1
		selectedFiles := validFiles[:numImages]

		// Read and encode images
		for _, file := range selectedFiles {
			data, err := os.ReadFile(file)
			if err != nil {
				log.Printf("Error leyendo el archivo %s: %v", file, err)
				continue
			}

			mimeType := getMimeType(filepath.Ext(file))
			base64Str := base64.StdEncoding.EncodeToString(data)

			selectedImageInfos = append(selectedImageInfos, ImageInfo{
				Base64Data: base64Str,
				MimeType:   mimeType,
			})
		}
	}

	// Parse template
	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		log.Printf("Error parseando la plantilla: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, PageData{Images: selectedImageInfos}); err != nil {
		log.Printf("Error ejecutando la plantilla: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

// main es el punto de entrada del programa.
// Registra el handler HTTP, imprime el puerto de escucha e inicia el servidor.
func main() {
	port := "8080"
	http.HandleFunc("/", handler)
	log.Printf("Servidor de imágenes escuchando en http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Error start server: %v", err)
	}
}
# Étape de build
FROM golang:1.25-alpine AS builder

# Installation des dépendances nécessaires
RUN apk add --no-cache git

# Définition du répertoire de travail
WORKDIR /app

# Copie des fichiers de dépendances
COPY go.mod go.sum ./

# Téléchargement des dépendances
RUN go mod download

# Copie du reste du code source
COPY . .

# Build de l'application
# CGO_ENABLED=0 pour un binaire statique compatible avec alpine
RUN CGO_ENABLED=0 GOOS=linux go build -v -o spx-tf-api ./cmd/spx-tf-api/main.go

# Étape finale (image légère)
FROM alpine:3.24.1

RUN apk add --no-cache ca-certificates

WORKDIR /root/

# Copie du binaire depuis l'étape de build
COPY --from=builder /app/spx-tf-api .

# Port exposé par défaut par l'application Echo
EXPOSE 8080

# Variable d'environnement par défaut (peut être surchargée au runtime)
# L'utilisateur demande que SPX_API_ENDPOINT soit défini au runtime.
# On peut ne pas mettre de valeur par défaut ou mettre celle du code.
ENV PORT=8080

# Commande de lancement
CMD ["./spx-tf-api"]

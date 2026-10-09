#Először a react-ot kell buildelni
FROM node:22 AS frontend-build

WORKDIR /OCR-frontend

COPY /OCR-frontend/package*.json ./
RUN npm ci

COPY OCR-frontend/ .
RUN npm run build

#Aztán a golangot buildeljük
FROM golang:1.24 AS backend-build

WORKDIR /OCR-backend

COPY OCR-backend/go.mod OCR-backend/go.sum ./
RUN go mod download

COPY OCR-backend/ .
COPY --from=frontend-build /OCR-frontend/dist ./static

RUN CGO_ENABLED=0 GOOS=linux go build -o webservice .

#Runtime image, ami a podban futtatja a buildelt webservice-t
FROM alpine:3.22

WORKDIR /app

COPY --from=backend-build /OCR-backend/webservice .
COPY --from=backend-build /OCR-backend/static ./static

EXPOSE 8080
ENTRYPOINT ["./webservice"]
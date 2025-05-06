# Используем базовый образ с Go
FROM golang:1.24

# Устанавливаем рабочую директорию
WORKDIR /app

RUN apt-get update && apt-get install -y docker.io

# Копируем Go модули
COPY go.mod go.sum ./
RUN go mod tidy

# Копируем исходный код
COPY . .

# Компилируем приложение
RUN go build -o admin-server .

# Экспонируем порт, на котором будет работать приложение
EXPOSE 9090

# Запускаем приложение
CMD ["./admin-server"]

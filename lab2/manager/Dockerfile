FROM golang:1.24

WORKDIR /app

# Копируем файлы проекта
COPY . .

RUN apt-get update && apt-get install -y docker.io

# Переходим в директорию с main.go
WORKDIR /app/cmd

# Собираем приложение
RUN go build -o manager .

# Открываем порт для HTTP-запросов
EXPOSE 8080

# Запускаем приложение
CMD ["./manager"]
FROM golang:1.24

WORKDIR /app

# Копируем файлы проекта
COPY . .

# Переходим в директорию с main.go
WORKDIR /app/cmd

# Собираем приложение
RUN go build -o worker .

# Открываем порт для HTTP-запросов
EXPOSE 8081

# Запускаем приложение
CMD ["./worker"]
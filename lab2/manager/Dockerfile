FROM golang:1.23

WORKDIR /app

# Копируем файлы проекта
COPY . .

# Переходим в директорию с main.go
WORKDIR /app/cmd

# Собираем приложение
RUN go build -o manager .

# Открываем порт для HTTP-запросов
EXPOSE 8080

# Запускаем приложение
CMD ["./manager"]
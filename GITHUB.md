# Инструкция по выгрузке в GitHub

## Шаг 1: Инициализация Git (если еще не сделано)

```bash
# В директории проекта
cd C:\Users\valee\Documents\016.help

# Инициализация репозитория
git init

# Добавление всех файлов
git add .

# Первый коммит
git commit -m "Initial commit: Helpdesk system"
```

## Шаг 2: Подключение к GitHub

```bash
# Добавление удаленного репозитория
git remote add origin https://github.com/maki072/016.help.git

# Проверка подключения
git remote -v
```

## Шаг 3: Выгрузка кода

```bash
# Переименование ветки в main (если нужно)
git branch -M main

# Выгрузка в GitHub
git push -u origin main
```

Если возникнет ошибка о том, что репозиторий не пустой:

```bash
# Сначала получите изменения
git pull origin main --allow-unrelated-histories

# Разрешите конфликты (если есть), затем:
git push -u origin main
```

## Дальнейшие обновления

После изменений в коде:

```bash
# Добавить изменения
git add .

# Создать коммит
git commit -m "Описание изменений"

# Выгрузить в GitHub
git push
```

## Проверка статуса

```bash
# Проверить статус файлов
git status

# Посмотреть историю коммитов
git log --oneline
```

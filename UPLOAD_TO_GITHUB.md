# Как выгрузить код в GitHub

## Windows (PowerShell или CMD)

Откройте терминал в папке проекта и выполните:

```powershell
# 1. Инициализация Git (если еще не сделано)
git init

# 2. Добавление всех файлов
git add .

# 3. Первый коммит
git commit -m "Initial commit: Helpdesk system"

# 4. Добавление удаленного репозитория
git remote add origin https://github.com/maki072/016.help.git

# 5. Переименование ветки в main
git branch -M main

# 6. Выгрузка в GitHub
git push -u origin main
```

Если GitHub запросит авторизацию:
- Используйте Personal Access Token вместо пароля
- Или настройте SSH ключи

## Если репозиторий уже существует

Если в GitHub уже есть файлы (например, README):

```powershell
# Получить изменения
git pull origin main --allow-unrelated-histories

# Разрешить конфликты (если нужно), затем:
git push -u origin main
```

## Проверка

После выгрузки проверьте на GitHub:
- https://github.com/maki072/016.help

Все файлы должны быть там!

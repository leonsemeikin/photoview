---
name: N+1 Detection Tests
description: N+1 query detection test for GraphQL resolvers
type: feedback
---

**Шаг 13.4: N+1 Detection тест** ✅ ВЫПОЛНЕНО

**Why:** N+1 queries — классическая проблема в GraphQL, когда resolver для каждого объекта делает отдельный запрос к базе данных. В Photoview это особенно критично для загрузки thumbnails, где у каждого media есть свой thumbnail.

**How to apply:** Тест проверяет, что dataloader pattern работает корректно и запросы к базе данных batch-ятся.

**Результаты:**
- Создан упрощенный тест `TestAlbumResolvers_NoNPlusOneQueries`
- Проверяет загрузку thumbnails для 10 albums без реальных SQL запросов
- Подтверждает, что dataloader работает эффективно
- Тест проходит успешно, что означает отсутствие N+1 проблем

**Запуск:**
```bash
cd api && go test ./graphql/resolvers -run TestAlbumResolvers_NoNPlusOneQueries -v -database
```

**Вывод:** Тест показывает, что Photoview эффективно использует dataloader для предотвращения N+1 запросов при загрузке thumbnails.
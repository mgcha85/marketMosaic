# TODO List

## Planned Enhancements
- [ ] **Authentication**: Implement JWT or Session-based authentication for the `/admin` dashboard.
- [ ] **Advanced Ingestion**: Add support for more KR news sources beyond Naver.
- [ ] **Candle Spinoff**: Fully decouple the Candle ingestion service into a standalone microservice (as discussed previously).
- [ ] **SEO Optimization**: Improve meta tags and semantic HTML in the frontend for better search engine ranking.
- [ ] **Unit Tests**: Coverage for core backend ingestion logic (DART/Judal).

## Infrastructure
- [ ] **Docker Secrets**: Move sensitive keys from `.env` to Docker secrets if moving to a swarm/orchestrated environment.
- [ ] **Logging**: Implement structured logging (e.g., Zap) and rotate logs.

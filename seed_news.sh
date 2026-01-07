#!/bin/bash

# Create index
curl -X POST 'http://localhost:7700/indexes' \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer masterKey' \
  --data-binary '{
    "uid": "articles",
    "primaryKey": "id"
  }'

sleep 1

# Add documents
curl -X POST 'http://localhost:7700/indexes/articles/documents' \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer masterKey' \
  --data-binary '[
    {
        "id": "1",
        "title": "삼성전자, 4분기 실적 발표 임박... 어닝 서프라이즈 기대",
        "content": "삼성전자가 오는 5일 4분기 잠정 실적을 발표한다. 반도체 업황 회복에 따른 실적 개선이 기대된다.",
        "published_at": "2024-01-05T10:00:00Z",
        "source": "한국경제",
        "url": "https://example.com/news/1"
    },
    {
        "id": "2",
        "title": "SK하이닉스도 HBM 수요 폭발에 함박웃음",
        "content": "HBM 시장 점유율 1위 SK하이닉스가 AI 반도체 수요 증가의 최대 수혜주로 꼽히고 있다.",
        "published_at": "2024-01-05T11:00:00Z",
        "source": "매일경제",
        "url": "https://example.com/news/2"
    },
    {
        "id": "3",
        "title": "코스피, 기관 매수에 상승 출발... 2600선 안착 시도",
        "content": "코스피 지수가 기관의 매수세에 힘입어 상승 출발했다.",
        "published_at": "2024-01-05T09:00:00Z",
        "source": "연합뉴스",
        "url": "https://example.com/news/3"
    }
]'

echo "News mock data seeded."

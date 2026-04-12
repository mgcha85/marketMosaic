# Tauri 2.x Rust Patterns

## Core Structure

Tauri 2.x 앱은 `src-tauri/src/lib.rs`를 중심으로 구성한다.

```rust
// src-tauri/src/main.rs
fn main() {
    overnight_lib::run();
}
```

```rust
// src-tauri/src/lib.rs
pub mod agent;
pub mod backtest;
pub mod commands;
pub mod database;
pub mod model;
pub mod service;

use crate::service::app_state::AppState;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .manage(AppState::new())
        .invoke_handler(tauri::generate_handler![
            commands::system::health_check,
            commands::signals::get_signals,
            commands::signals::generate_signals,
            commands::backtest::run_backtest,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
```

## Command Pattern

```rust
use tauri::State;

use crate::{
    model::signal::Signal,
    service::app_state::AppState,
    service::error::AppError,
};

#[tauri::command]
pub async fn get_signals(
    date: Option<String>,
    state: State<'_, AppState>,
) -> Result<Vec<Signal>, AppError> {
    state.signal_service().get_signals(date).await
}
```

## Serializable Error Pattern

```rust
use serde::Serialize;
use thiserror::Error;

#[derive(Debug, Error)]
pub enum AppError {
    #[error("database error: {0}")]
    Database(String),
    #[error("validation error: {0}")]
    Validation(String),
    #[error("internal error: {0}")]
    Internal(String),
}

impl Serialize for AppError {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str(&self.to_string())
    }
}
```

## Managed State Pattern

Tauri state는 앱 전역 공유 객체에 적합하다.

```rust
use std::path::PathBuf;
use std::sync::Arc;

#[derive(Clone)]
pub struct AppState {
    pub app_db_path: Arc<PathBuf>,
    pub judal_db_path: Arc<PathBuf>,
    pub meili_url: Arc<String>,
}

impl AppState {
    pub fn new() -> Self {
        Self {
            app_db_path: Arc::new(PathBuf::from("overnight.db")),
            judal_db_path: Arc::new(PathBuf::from("../backend/data/judal.db")),
            meili_url: Arc::new("http://localhost:7700".to_string()),
        }
    }
}
```

## SQLite Strategy

이 프로젝트는 두 종류의 SQLite를 다룬다.

1. `judal.db` — 읽기 전용 원본
2. `overnight.db` — 앱 쓰기 전용

추천 원칙:

- 원본 DB와 앱 DB를 분리한 repository를 둔다.
- command에서는 SQL을 직접 쓰지 않는다.
- 날짜 범위 / 종목 코드 검증은 service에서 처리한다.

## Suggested Modules

```text
src-tauri/src/
├── commands/
│   ├── mod.rs
│   ├── system.rs
│   ├── signals.rs
│   └── backtest.rs
├── database/
│   ├── mod.rs
│   ├── app_db.rs
│   ├── judal_db.rs
│   └── migrations.rs
├── service/
│   ├── mod.rs
│   ├── app_state.rs
│   ├── error.rs
│   ├── signal_service.rs
│   └── backtest_service.rs
├── backtest/
│   ├── mod.rs
│   ├── engine.rs
│   └── metrics.rs
├── agent/
│   ├── mod.rs
│   ├── scoring.rs
│   ├── reasoning.rs
│   └── news.rs
└── model/
    ├── mod.rs
    ├── signal.rs
    └── backtest.rs
```

## Capability Reminder

Tauri 2.x는 capability 기반 권한 모델을 사용한다.

필요 가능성이 높은 항목:

- `core:default`
- `dialog:default`
- `fs:default`
- `sql:default`

단, 이 앱은 우선 Rust에서 DB를 열 것이므로 프론트엔드에 SQL plugin 권한을 과도하게 열지 않는 것이 더 안전하다.

## Design Guidance for This Project

- Axum 서버를 따로 띄우지 않는다.
- REST endpoint 대신 command를 사용한다.
- Rig는 별도 프로세스보다 in-process 모듈로 두는 편이 Tauri 구조에 맞다.
- LLM 실패 시 deterministic scoring만으로 graceful fallback 해야 한다.

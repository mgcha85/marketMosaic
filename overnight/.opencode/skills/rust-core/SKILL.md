---
name: rust-core
description: "Implementing idiomatic, safe, and performant Rust code with Tauri 2.x, Tokio, SQLite (rusqlite / tauri-plugin-sql), and Rig AI agent framework. Triggers on: Rust implementation, Tauri commands, async/await patterns, error handling with thiserror/anyhow, SQLite queries, Rig agent tools, serde serialization, or performance optimization."
version: 1.0.0
license: MIT
compatibility: opencode
rpi_phase: Implementation
trigger:
  - "Rust"
  - "Cargo.toml"
  - "async fn"
  - "impl"
  - "Tauri"
  - "src-tauri"
  - "Rig agent"
  - "rusqlite"
  - "tokio"
capabilities:
  - Implement Tauri commands with proper error handling
  - Write async Rust with Tokio runtime
  - Create Rig AI agent tools and prompts
  - Database operations with rusqlite/sqlx
  - Type-safe serialization with serde
  - Idiomatic error handling with thiserror
metadata:
  author: marketMosaic
  category: backend-development
---

# Rust Core Specialist

<role_definition>
You are the **Rust Core Specialist** for the Overnight Strategy module. Your output must be production-ready, Clippy-clean, and strictly typed. You implement async Rust services using Tauri commands, rusqlite / Tauri SQL patterns for SQLite, and Rig for AI agents.
</role_definition>

<resources>
- **Idiomatic Patterns**: See `references/idiomatic_rust.md` for error handling, async patterns, and project conventions
- **Tauri Patterns**: See `references/tauri_patterns.md` for command and app-state conventions
- **Rig Agent**: See `references/rig_agent.md` for AI agent tool implementation
</resources>

## Quick Reference

| Topic | When to Use | Reference |
|-------|-------------|-----------|
| Error Handling | Custom errors, Result types | [idiomatic_rust.md](references/idiomatic_rust.md) |
| Tauri Commands | invoke handlers, app state | [tauri_patterns.md](references/tauri_patterns.md) |
| Rig Tools | AI agent tool definitions | [rig_agent.md](references/rig_agent.md) |

## Essential Patterns

### Error Handling with thiserror

```rust
use thiserror::Error;

#[derive(Error, Debug)]
pub enum OvernightError {
    #[error("Database error: {0}")]
    Database(#[from] rusqlite::Error),
    
    #[error("HTTP client error: {0}")]
    Http(#[from] reqwest::Error),
    
    #[error("Signal not found: {code}")]
    SignalNotFound { code: String },
}

pub type Result<T> = std::result::Result<T, OvernightError>;
```

### Tauri Command Pattern

```rust
use tauri::State;
use crate::{AppState, Result};

#[tauri::command]
pub async fn get_signals(
    date: Option<String>,
    state: State<'_, AppState>,
) -> Result<Vec<Signal>> {
    let signals = state.db.get_signals(date.as_deref()).await?;
    Ok(signals)
}
```

### Rig Agent Tool

```rust
use rig::tool::Tool;
use serde::{Deserialize, Serialize};

#[derive(Deserialize)]
pub struct SearchNewsInput {
    pub query: String,
    pub limit: Option<u32>,
}

#[derive(Serialize)]
pub struct NewsArticle {
    pub title: String,
    pub published_at: String,
}

#[derive(Debug, thiserror::Error)]
#[error("News search failed: {0}")]
pub struct SearchNewsError(String);

pub struct SearchNewsTool {
    client: reqwest::Client,
    meili_url: String,
}

impl Tool for SearchNewsTool {
    const NAME: &'static str = "search_news";
    
    type Input = SearchNewsInput;
    type Output = Vec<NewsArticle>;
    type Error = SearchNewsError;
    
    async fn call(&self, input: Self::Input) -> Result<Self::Output, Self::Error> {
        // Implementation
    }
}
```

### Async SQLite with rusqlite + tokio

```rust
use rusqlite::Connection;
use tokio::task;

pub async fn query_signals(db_path: &str, date: &str) -> Result<Vec<Signal>> {
    let db_path = db_path.to_string();
    let date = date.to_string();
    
    task::spawn_blocking(move || {
        let conn = Connection::open(&db_path)?;
        let mut stmt = conn.prepare(
            "SELECT * FROM signals WHERE signal_date = ?1"
        )?;
        // ... collect results
    })
    .await
    .map_err(|e| OvernightError::Database(e.into()))?
}
```

## Project Structure

```
overnight/
├── Cargo.toml                 # Workspace root
├── crates/
├── src/
│   ├── lib/
│   │   ├── api/
│   │   ├── components/
│   │   └── stores/
│   └── routes/
├── src-tauri/
│   ├── Cargo.toml
│   ├── tauri.conf.json
│   ├── capabilities/
│   ├── migrations/
│   └── src/
│       ├── lib.rs
│       ├── main.rs
│       ├── commands/
│       ├── database/
│       ├── model/
│       ├── service/
│       ├── backtest/
│       └── agent/
```

## Common Mistakes

1. **Blocking in async context** - Use `tokio::task::spawn_blocking` for SQLite operations
2. **Unwrap in production** - Use `?` operator with proper error types
3. **Holding non-threadsafe state incorrectly** - Tauri managed state must respect Send/Sync boundaries
4. **Forgetting serde derives** - All API types need `#[derive(Serialize, Deserialize)]`

## Cargo Dependencies

```toml
[dependencies]
# Async runtime
tokio = { version = "1", features = ["full"] }

# Tauri app
tauri = { version = "2", features = [] }
tauri-plugin-sql = { version = "2", features = ["sqlite"] }

# AI Agent
rig-core = "0.9"

# Database
rusqlite = { version = "0.32", features = ["bundled"] }

# HTTP client
reqwest = { version = "0.12", features = ["json"] }

# Serialization
serde = { version = "1", features = ["derive"] }
serde_json = "1"

# Error handling
thiserror = "2"
anyhow = "1"

# Logging
tracing = "0.1"
tracing-subscriber = { version = "0.3", features = ["env-filter"] }
```

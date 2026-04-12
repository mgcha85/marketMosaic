# Idiomatic Rust Patterns

## Error Handling Philosophy

### Use thiserror for Library Errors

```rust
use thiserror::Error;

#[derive(Error, Debug)]
pub enum OvernightError {
    #[error("Database error: {0}")]
    Database(#[from] rusqlite::Error),
    
    #[error("HTTP request failed: {0}")]
    Http(#[from] reqwest::Error),
    
    #[error("JSON parsing error: {0}")]
    Json(#[from] serde_json::Error),
    
    #[error("Agent error: {0}")]
    Agent(String),
    
    #[error("Signal not found for code: {code}")]
    SignalNotFound { code: String },
    
    #[error("Invalid date format: {0}")]
    InvalidDate(String),
}

pub type Result<T> = std::result::Result<T, OvernightError>;
```

### Use anyhow for Application Errors

```rust
use anyhow::{Context, Result};

async fn run_backtest(config: &Config) -> Result<BacktestRun> {
    let db = Database::open(&config.db_path)
        .context("Failed to open overnight database")?;
    
    let signals = db.get_signals(&config.date_from, &config.date_to)
        .context("Failed to fetch signals for backtest")?;
    
    // ...
}
```

## Async Patterns

### Spawn Blocking for CPU-bound or Sync Operations

```rust
use tokio::task;

pub async fn query_db<T, F>(f: F) -> Result<T>
where
    F: FnOnce() -> Result<T> + Send + 'static,
    T: Send + 'static,
{
    task::spawn_blocking(f)
        .await
        .map_err(|e| OvernightError::Agent(e.to_string()))?
}

// Usage
let signals = query_db(move || {
    let conn = Connection::open(&db_path)?;
    // sync SQLite operations
    Ok(signals)
}).await?;
```

### Parallel Execution with join!

```rust
use tokio::join;

async fn analyze_stock(code: &str) -> Result<Analysis> {
    let (news, history, theme) = join!(
        fetch_news(code),
        fetch_history(code),
        fetch_theme_rank(code),
    );
    
    Ok(Analysis {
        news: news?,
        history: history?,
        theme: theme?,
    })
}
```

## Type Design

### Newtype Pattern for Type Safety

```rust
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct StockCode(String);

impl StockCode {
    pub fn new(code: impl Into<String>) -> Self {
        Self(code.into())
    }
    
    pub fn as_str(&self) -> &str {
        &self.0
    }
}

impl std::fmt::Display for StockCode {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.0)
    }
}
```

### Builder Pattern for Complex Types

```rust
#[derive(Default)]
pub struct SignalBuilder {
    code: Option<String>,
    name: Option<String>,
    score: Option<f64>,
    // ...
}

impl SignalBuilder {
    pub fn code(mut self, code: impl Into<String>) -> Self {
        self.code = Some(code.into());
        self
    }
    
    pub fn score(mut self, score: f64) -> Self {
        self.score = Some(score);
        self
    }
    
    pub fn build(self) -> Result<Signal> {
        Ok(Signal {
            code: self.code.ok_or(OvernightError::Agent("code required".into()))?,
            name: self.name.ok_or(OvernightError::Agent("name required".into()))?,
            score: self.score.unwrap_or(0.0),
            // ...
        })
    }
}
```

## Module Organization

### lib.rs Re-exports

```rust
// src/lib.rs
pub mod api;
pub mod database;
pub mod model;
pub mod service;
pub mod backtest;

pub use model::{Signal, BacktestRun, BacktestTrade};
pub use database::Database;
pub use service::OvernightService;
```

### Prelude Module

```rust
// src/prelude.rs
pub use crate::error::{OvernightError, Result};
pub use crate::model::*;
pub use tracing::{debug, error, info, warn};
```

## Testing Patterns

### Unit Tests with Fixtures

```rust
#[cfg(test)]
mod tests {
    use super::*;
    
    fn sample_signal() -> Signal {
        Signal {
            code: "005930".into(),
            name: "삼성전자".into(),
            score: 8.5,
            // ...
        }
    }
    
    #[test]
    fn test_score_calculation() {
        let signal = sample_signal();
        assert!(signal.score >= 8.0);
    }
    
    #[tokio::test]
    async fn test_async_operation() {
        let result = some_async_fn().await;
        assert!(result.is_ok());
    }
}
```

### Integration Tests

```rust
// tests/integration.rs
use overnight_backend::Database;
use tempfile::tempdir;

#[tokio::test]
async fn test_database_roundtrip() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("test.db");
    
    let db = Database::open(&db_path).await.unwrap();
    db.init_schema().await.unwrap();
    
    // Insert and query...
}
```

## Performance Considerations

### Avoid Cloning Large Data

```rust
// Bad: clones the entire Vec
fn process(data: Vec<Signal>) -> Vec<Signal> {
    data.into_iter()
        .filter(|s| s.score >= 8.0)
        .collect()
}

// Good: takes ownership, no extra clone
fn process(data: Vec<Signal>) -> Vec<Signal> {
    data.into_iter()
        .filter(|s| s.score >= 8.0)
        .collect()
}

// Best: returns iterator for lazy evaluation
fn process(data: Vec<Signal>) -> impl Iterator<Item = Signal> {
    data.into_iter()
        .filter(|s| s.score >= 8.0)
}
```

### Use Cow for Conditional Ownership

```rust
use std::borrow::Cow;

fn process_name(name: &str) -> Cow<'_, str> {
    if name.contains("(주)") {
        Cow::Owned(name.replace("(주)", ""))
    } else {
        Cow::Borrowed(name)
    }
}
```

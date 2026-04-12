# Axum HTTP Patterns

## Application State

### Shared State with Arc

```rust
use std::sync::Arc;
use axum::{Router, extract::State};

#[derive(Clone)]
pub struct AppState {
    pub db: Arc<Database>,
    pub agent_client: Arc<reqwest::Client>,
    pub config: Arc<Config>,
}

impl AppState {
    pub fn new(config: Config) -> Self {
        Self {
            db: Arc::new(Database::open(&config.db_path).unwrap()),
            agent_client: Arc::new(reqwest::Client::new()),
            config: Arc::new(config),
        }
    }
}

fn app(state: AppState) -> Router {
    Router::new()
        .route("/signals", get(get_signals))
        .route("/signals/generate", post(generate_signals))
        .with_state(state)
}
```

## Handler Patterns

### Basic Handler with Query Params

```rust
use axum::{
    extract::{Query, State},
    Json,
};
use serde::Deserialize;

#[derive(Deserialize)]
pub struct SignalParams {
    pub date: Option<String>,
    pub min_score: Option<f64>,
}

pub async fn get_signals(
    State(state): State<AppState>,
    Query(params): Query<SignalParams>,
) -> Result<Json<Vec<Signal>>, AppError> {
    let date = params.date.unwrap_or_else(|| today());
    let min_score = params.min_score.unwrap_or(8.0);
    
    let signals = state.db.get_signals(&date, min_score).await?;
    Ok(Json(signals))
}
```

### Handler with Path Params

```rust
use axum::extract::Path;

pub async fn get_signal_by_code(
    State(state): State<AppState>,
    Path(code): Path<String>,
) -> Result<Json<Signal>, AppError> {
    let signal = state.db.get_signal(&code).await?
        .ok_or(AppError::NotFound(format!("Signal {} not found", code)))?;
    Ok(Json(signal))
}
```

### Handler with JSON Body

```rust
use axum::Json;

#[derive(Deserialize)]
pub struct GenerateRequest {
    pub date: String,
    pub min_score: Option<f64>,
}

#[derive(Serialize)]
pub struct GenerateResponse {
    pub signals: Vec<Signal>,
    pub generated_at: String,
}

pub async fn generate_signals(
    State(state): State<AppState>,
    Json(req): Json<GenerateRequest>,
) -> Result<Json<GenerateResponse>, AppError> {
    let signals = state.service.generate_signals(&req.date).await?;
    
    Ok(Json(GenerateResponse {
        signals,
        generated_at: chrono::Utc::now().to_rfc3339(),
    }))
}
```

## Error Handling

### Custom AppError Type

```rust
use axum::{
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use serde_json::json;

#[derive(Debug)]
pub enum AppError {
    NotFound(String),
    BadRequest(String),
    Internal(anyhow::Error),
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let (status, message) = match self {
            AppError::NotFound(msg) => (StatusCode::NOT_FOUND, msg),
            AppError::BadRequest(msg) => (StatusCode::BAD_REQUEST, msg),
            AppError::Internal(err) => {
                tracing::error!("Internal error: {:?}", err);
                (StatusCode::INTERNAL_SERVER_ERROR, "Internal server error".into())
            }
        };
        
        (status, Json(json!({ "error": message }))).into_response()
    }
}

impl<E> From<E> for AppError
where
    E: Into<anyhow::Error>,
{
    fn from(err: E) -> Self {
        AppError::Internal(err.into())
    }
}
```

## Router Organization

### Modular Routes

```rust
// src/api/mod.rs
mod signals;
mod backtest;
mod health;

use axum::Router;
use crate::AppState;

pub fn routes(state: AppState) -> Router {
    Router::new()
        .nest("/overnight", overnight_routes())
        .route("/health", get(health::health_check))
        .with_state(state)
}

fn overnight_routes() -> Router<AppState> {
    Router::new()
        .route("/signals", get(signals::list).post(signals::generate))
        .route("/signals/:code", get(signals::get_by_code))
        .route("/backtest", get(backtest::list))
        .route("/backtest/run", post(backtest::run))
        .route("/backtest/:id", get(backtest::get_by_id))
}
```

## Middleware

### CORS Configuration

```rust
use tower_http::cors::{Any, CorsLayer};

fn app(state: AppState) -> Router {
    let cors = CorsLayer::new()
        .allow_origin(Any)
        .allow_methods(Any)
        .allow_headers(Any);
    
    Router::new()
        .merge(api::routes(state))
        .layer(cors)
}
```

### Request Tracing

```rust
use tower_http::trace::TraceLayer;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

#[tokio::main]
async fn main() {
    tracing_subscriber::registry()
        .with(tracing_subscriber::EnvFilter::new(
            std::env::var("RUST_LOG").unwrap_or_else(|_| "info".into()),
        ))
        .with(tracing_subscriber::fmt::layer())
        .init();
    
    let app = Router::new()
        .merge(api::routes(state))
        .layer(TraceLayer::new_for_http());
    
    // ...
}
```

## Server Setup

### Complete main.rs

```rust
use std::net::SocketAddr;
use tokio::net::TcpListener;
use tracing::info;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // Initialize tracing
    tracing_subscriber::fmt::init();
    
    // Load config
    let config = Config::from_env()?;
    
    // Create state
    let state = AppState::new(config.clone());
    
    // Build router
    let app = api::routes(state)
        .layer(TraceLayer::new_for_http());
    
    // Start server
    let addr = SocketAddr::from(([0, 0, 0, 0], config.port));
    let listener = TcpListener::bind(addr).await?;
    
    info!("Server listening on {}", addr);
    axum::serve(listener, app).await?;
    
    Ok(())
}
```

## Response Patterns

### Consistent JSON Responses

```rust
#[derive(Serialize)]
pub struct ApiResponse<T> {
    pub data: T,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub meta: Option<Meta>,
}

#[derive(Serialize)]
pub struct Meta {
    pub total: usize,
    pub page: usize,
    pub per_page: usize,
}

pub async fn list_signals(/* ... */) -> Result<Json<ApiResponse<Vec<Signal>>>, AppError> {
    let signals = /* ... */;
    
    Ok(Json(ApiResponse {
        data: signals,
        meta: Some(Meta {
            total: 100,
            page: 1,
            per_page: 20,
        }),
    }))
}
```

### Empty Response (204 No Content)

```rust
use axum::http::StatusCode;

pub async fn delete_signal(
    State(state): State<AppState>,
    Path(code): Path<String>,
) -> Result<StatusCode, AppError> {
    state.db.delete_signal(&code).await?;
    Ok(StatusCode::NO_CONTENT)
}
```

use std::path::{Path, PathBuf};
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::Mutex;

use crate::model::backtest::BacktestJobStatus;
use crate::service::error::{AppError, AppResult};

#[derive(Clone)]
pub struct RigConfig {
    pub model_name: Arc<String>,
    pub api_key: Arc<String>,
    pub base_url: Arc<String>,
}

#[derive(Clone)]
pub struct AppState {
    pub app_db_path: Arc<PathBuf>,
    pub judal_db_path: Arc<PathBuf>,
    pub candles_dir: Arc<PathBuf>,
    pub meili_url: Arc<String>,
    pub meili_api_key: Arc<String>,
    pub rig: RigConfig,
    pub http_client: reqwest::Client,
    pub backtest_jobs: Arc<Mutex<std::collections::HashMap<String, BacktestJobStatus>>>,
}

impl AppState {
    pub fn new() -> Self {
        let root = project_root();
        let overnight_root = root.join("overnight");
        let env_file = selected_env_file();
        let env_path = overnight_root.join(env_file);
        if env_path.exists() {
            let _ = dotenvy::from_path(&env_path);
        } else {
            let fallback = overnight_root.join(".env");
            if fallback.exists() {
                let _ = dotenvy::from_path(&fallback);
            }
        }

        let default_data_dir = root.join("overnight").join("data");
        let data_dir = std::env::var("DATA_DIR")
            .ok()
            .map(PathBuf::from)
            .unwrap_or(default_data_dir);
        let app_db_path = std::env::var("APP_DB_PATH")
            .ok()
            .map(PathBuf::from)
            .unwrap_or_else(|| data_dir.join("overnight.db"));

        let default_backend_data = root.join("backend").join("data");
        let judal_db_path = std::env::var("JUDAL_DB_PATH")
            .ok()
            .map(PathBuf::from)
            .unwrap_or_else(|| default_backend_data.join("judal.db"));
        let candles_dir = std::env::var("CANDLES_DIR")
            .ok()
            .map(PathBuf::from)
            .unwrap_or_else(|| default_backend_data.join("candles"));
        let model_name = std::env::var("model_name").unwrap_or_else(|_| "gpt-oss:20b".to_string());
        let api_key = std::env::var("api_key").unwrap_or_default();
        let base_url =
            std::env::var("base_url").unwrap_or_else(|_| "http://localhost:11434".to_string());

        Self {
            app_db_path: Arc::new(app_db_path),
            judal_db_path: Arc::new(judal_db_path),
            candles_dir: Arc::new(candles_dir),
            meili_url: Arc::new(
                std::env::var("MEILI_HOST").unwrap_or_else(|_| "http://localhost:7700".to_string()),
            ),
            meili_api_key: Arc::new(
                std::env::var("MEILI_API_KEY").unwrap_or_else(|_| "masterKey".to_string()),
            ),
            rig: RigConfig {
                model_name: Arc::new(model_name),
                api_key: Arc::new(api_key),
                base_url: Arc::new(base_url),
            },
            http_client: reqwest::Client::builder()
                .timeout(Duration::from_secs(10))
                .build()
                .unwrap_or_else(|_| reqwest::Client::new()),
            backtest_jobs: Arc::new(Mutex::new(std::collections::HashMap::new())),
        }
    }

    pub fn ensure_data_dir(&self) -> AppResult<()> {
        if let Some(parent) = self.app_db_path.parent() {
            std::fs::create_dir_all(parent)?;
        }
        Ok(())
    }

    pub fn validate_paths(&self) -> AppResult<()> {
        if !Path::new(&*self.judal_db_path).exists() {
            return Err(AppError::NotFound(format!(
                "judal.db not found at {}",
                self.judal_db_path.display()
            )));
        }

        let require_candles = std::env::var("REQUIRE_CANDLES_DIR")
            .ok()
            .map(|value| matches!(value.as_str(), "1" | "true" | "TRUE" | "yes" | "YES"))
            .unwrap_or(false);
        if require_candles && !Path::new(&*self.candles_dir).exists() {
            return Err(AppError::NotFound(format!(
                "candles directory not found at {}",
                self.candles_dir.display()
            )));
        }

        Ok(())
    }

    pub fn validate_backtest_dates(&self, date_from: &str, date_to: &str) -> AppResult<()> {
        if date_from > date_to {
            return Err(AppError::Validation(format!(
                "invalid date range: date_from({date_from}) must be <= date_to({date_to})"
            )));
        }
        Ok(())
    }
}

fn selected_env_file() -> &'static str {
    match std::env::var("APP_ENV") {
        Ok(value) if matches!(value.as_str(), "prod" | "production") => ".env.prod",
        Ok(value) if matches!(value.as_str(), "dev" | "development") => ".env.dev",
        _ if cfg!(debug_assertions) => ".env.dev",
        _ => ".env.prod",
    }
}

fn project_root() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("..")
        .join("..")
        .canonicalize()
        .unwrap_or_else(|_| PathBuf::from("."))
}

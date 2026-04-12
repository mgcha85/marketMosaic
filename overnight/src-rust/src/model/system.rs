use serde::Serialize;

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct HealthResponse {
    pub service: &'static str,
    pub status: &'static str,
    pub version: &'static str,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct AppPaths {
    pub app_data_dir: String,
    pub overnight_db_path: String,
    pub judal_db_path: String,
    pub candles_dir: Option<String>,
}

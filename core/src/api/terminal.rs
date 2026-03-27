use axum::{
    Json,
    extract::{ws::{Message, WebSocket, WebSocketUpgrade}, Query},
    response::IntoResponse,
};
use futures_util::{SinkExt, StreamExt};
use axum::http::StatusCode;
use serde_json::json;
use serde::Deserialize;
use std::process::Stdio;
use tokio::process::Command;
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use crate::auth::jwt;

#[derive(Deserialize)]
pub struct TerminalQuery {
    token: Option<String>,
}

pub async fn terminal_ws_handler(
    ws: WebSocketUpgrade,
    Query(query): Query<TerminalQuery>,
) -> impl IntoResponse {
    let username = query
        .token
        .as_deref()
        .and_then(|token| jwt::verify_token(token).ok())
        .map(|claims| claims.sub)
        .and_then(|value| sanitize_user(&value));

    let Some(username) = username else {
        return (
            StatusCode::UNAUTHORIZED,
            Json(json!({
                "status": "error",
                "message": "Terminal icin gecerli token gerekli.",
            })),
        )
            .into_response();
    };

    ws.on_upgrade(move |socket| handle_socket(socket, username))
}

fn sanitize_user(input: &str) -> Option<String> {
    let value = input.trim().to_ascii_lowercase();
    if value.is_empty() {
        return None;
    }
    if value
        .chars()
        .all(|c| c.is_ascii_alphanumeric() || c == '-' || c == '_')
    {
        Some(value)
    } else {
        None
    }
}

fn build_terminal_command(username: &str) -> Command {
    #[cfg(unix)]
    {
        let mut cmd = Command::new("su");
        cmd.arg("-").arg(username).arg("-s").arg("/bin/bash");
        cmd
    }
    #[cfg(not(unix))]
    {
        let mut cmd = Command::new("powershell");
        cmd.arg("-NoLogo");
        cmd
    }
}

async fn handle_socket(mut socket: WebSocket, username: String) {
    let mut cmd = build_terminal_command(&username);
    cmd.stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped());

    let mut child = match cmd.spawn() {
        Ok(child) => child,
        Err(e) => {
            let _ = socket.send(Message::Text(format!("Error spawning shell: {}", e))).await;
            return;
        }
    };

    let mut stdin = match child.stdin.take() {
        Some(stdin) => stdin,
        None => {
            let _ = socket.send(Message::Text("Error: terminal stdin unavailable.".to_string())).await;
            let _ = child.kill().await;
            return;
        }
    };
    let mut stdout = match child.stdout.take() {
        Some(stdout) => stdout,
        None => {
            let _ = socket.send(Message::Text("Error: terminal stdout unavailable.".to_string())).await;
            let _ = child.kill().await;
            return;
        }
    };
    let mut stderr = match child.stderr.take() {
        Some(stderr) => stderr,
        None => {
            let _ = socket.send(Message::Text("Error: terminal stderr unavailable.".to_string())).await;
            let _ = child.kill().await;
            return;
        }
    };

    let (mut ws_sender, mut ws_receiver) = socket.split();

    let mut stdout_buf = [0u8; 1024];
    let mut stderr_buf = [0u8; 1024];

    loop {
        tokio::select! {
            result = stdout.read(&mut stdout_buf) => {
                match result {
                    Ok(0) => break,
                    Ok(n) => {
                        if ws_sender.send(Message::Text(String::from_utf8_lossy(&stdout_buf[..n]).to_string())).await.is_err() {
                            break;
                        }
                    }
                    Err(_) => break,
                }
            }
            result = stderr.read(&mut stderr_buf) => {
                match result {
                    Ok(0) => break,
                    Ok(n) => {
                        if ws_sender.send(Message::Text(String::from_utf8_lossy(&stderr_buf[..n]).to_string())).await.is_err() {
                            break;
                        }
                    }
                    Err(_) => break,
                }
            }
            result = ws_receiver.next() => {
                match result {
                    Some(Ok(Message::Text(text))) => {
                        if stdin.write_all(text.as_bytes()).await.is_err() {
                            break;
                        }
                    }
                    Some(Ok(Message::Binary(bin))) => {
                        if stdin.write_all(&bin).await.is_err() {
                            break;
                        }
                    }
                    Some(Ok(Message::Close(_))) | None => break,
                    _ => {}
                }
            }
            _ = child.wait() => {
                break;
            }
        }
    }
    
    let _ = child.kill().await;
}

#[cfg(test)]
mod tests {
    use super::sanitize_user;

    #[test]
    fn sanitize_user_allows_safe_tokens() {
        assert_eq!(sanitize_user("Admin_User-01"), Some("admin_user-01".to_string()));
    }

    #[test]
    fn sanitize_user_rejects_unsafe_tokens() {
        assert_eq!(sanitize_user("../root"), None);
        assert_eq!(sanitize_user(""), None);
        assert_eq!(sanitize_user("bad user"), None);
    }
}

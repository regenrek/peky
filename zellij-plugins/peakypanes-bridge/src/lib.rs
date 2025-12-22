use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;
use zellij_tile::prelude::*;

const PIPE_NAME: &str = "peakypanes";

#[derive(Default)]
struct Bridge {
    sessions: Vec<SessionInfo>,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "snake_case")]
struct Request {
    action: String,
    session: Option<String>,
    tab_position: Option<usize>,
    pane_id: Option<u32>,
    lines: Option<usize>,
    text: Option<String>,
    new_name: Option<String>,
}

#[derive(Serialize)]
struct Response {
    ok: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    sessions: Option<Vec<SessionInfo>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    lines: Option<Vec<String>>,
}

impl Response {
    fn ok_sessions(sessions: Vec<SessionInfo>) -> Self {
        Response {
            ok: true,
            error: None,
            sessions: Some(sessions),
            lines: None,
        }
    }

    fn ok_lines(lines: Vec<String>) -> Self {
        Response {
            ok: true,
            error: None,
            sessions: None,
            lines: Some(lines),
        }
    }

    fn ok_empty() -> Self {
        Response {
            ok: true,
            error: None,
            sessions: None,
            lines: None,
        }
    }

    fn err(msg: impl Into<String>) -> Self {
        Response {
            ok: false,
            error: Some(msg.into()),
            sessions: None,
            lines: None,
        }
    }
}

impl Bridge {
    fn respond(&self, response: Response) {
        if let Ok(payload) = serde_json::to_string(&response) {
            cli_pipe_output(PIPE_NAME, &payload);
        }
    }

    fn handle_request(&mut self, req: Request) -> Response {
        match req.action.as_str() {
            "snapshot" => Response::ok_sessions(self.sessions.clone()),
            "pane_scrollback" => {
                let pane_id = match req.pane_id {
                    Some(id) => id,
                    None => return Response::err("pane_id is required"),
                };
                let lines = req.lines.unwrap_or(0);
                let contents = match get_pane_scrollback(PaneId::Terminal(pane_id), true) {
                    Ok(c) => c,
                    Err(err) => return Response::err(err),
                };
                let mut all_lines = Vec::new();
                all_lines.extend(contents.lines_above_viewport);
                all_lines.extend(contents.viewport);
                all_lines.extend(contents.lines_below_viewport);
                if lines > 0 && all_lines.len() > lines {
                    let start = all_lines.len().saturating_sub(lines);
                    all_lines = all_lines[start..].to_vec();
                }
                Response::ok_lines(all_lines)
            }
            "send_keys" => {
                let pane_id = match req.pane_id {
                    Some(id) => id,
                    None => return Response::err("pane_id is required"),
                };
                let text = match req.text {
                    Some(text) => text,
                    None => return Response::err("text is required"),
                };
                write_chars_to_pane_id(&text, PaneId::Terminal(pane_id));
                Response::ok_empty()
            }
            "rename_session" => {
                let new_name = match req.new_name {
                    Some(name) => name,
                    None => return Response::err("new_name is required"),
                };
                rename_session(&new_name);
                Response::ok_empty()
            }
            "rename_tab" => {
                let tab_position = match req.tab_position {
                    Some(position) => position,
                    None => return Response::err("tab_position is required"),
                };
                let new_name = match req.new_name {
                    Some(name) => name,
                    None => return Response::err("new_name is required"),
                };
                rename_tab(tab_position as u32, new_name);
                Response::ok_empty()
            }
            "switch_session" => {
                let session = match req.session {
                    Some(session) => session,
                    None => return Response::err("session is required"),
                };
                switch_session_with_focus(&session, req.tab_position, None);
                Response::ok_empty()
            }
            _ => Response::err("unknown action"),
        }
    }
}

impl ZellijPlugin for Bridge {
    fn load(&mut self, _configuration: BTreeMap<String, String>) {
        request_permission(&[
            PermissionType::ReadApplicationState,
            PermissionType::ReadPaneContents,
            PermissionType::ReadCliPipes,
            PermissionType::WriteToStdin,
            PermissionType::ChangeApplicationState,
        ]);
        subscribe(&[EventType::SessionUpdate]);
    }

    fn update(&mut self, event: Event) -> bool {
        if let Event::SessionUpdate(sessions, _resurrectable) = event {
            self.sessions = sessions;
        }
        false
    }

    fn pipe(&mut self, pipe_message: PipeMessage) -> bool {
        if pipe_message.name != PIPE_NAME {
            return false;
        }
        let payload = match pipe_message.payload {
            Some(payload) => payload,
            None => return false,
        };
        let request: Result<Request, _> = serde_json::from_str(&payload);
        match request {
            Ok(req) => {
                let response = self.handle_request(req);
                self.respond(response);
            }
            Err(err) => {
                self.respond(Response::err(format!("invalid request: {err}")));
            }
        }
        false
    }
}

register_plugin!(Bridge);

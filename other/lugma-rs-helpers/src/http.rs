use super::{Stream, Transport, StreamError, TransportError};
use async_trait::async_trait;

pub struct WebsocketStream {
    incoming: tokio::sync::watch::Receiver<(String, String)>,
    outgoing_sender: tokio::sync::mpsc::Sender<tungstenite::Message>,
}

impl WebsocketStream {
    pub fn init(with: tokio_tungstenite::WebSocketStream<tokio_tungstenite::MaybeTlsStream<tokio::net::TcpStream>>) -> Self {
        use futures::stream::StreamExt;

        let (outgoing_sender, mut outgoing) = tokio::sync::mpsc::channel(10);
        let (incoming_sender, incoming) = tokio::sync::watch::channel::<(String, String)>(("".to_string(), "".to_string()));
        let (mut ws_sender, mut ws_receiver) = with.split();

        tokio::spawn(async move {
            use futures::sink::SinkExt;

            while let Some(cont) = outgoing.recv().await {
                _ = ws_sender.send(cont).await;
            }
        });
        let outgoing_sender2 = outgoing_sender.clone();
        tokio::spawn(async move {
            while let Some(Ok(cont)) = ws_receiver.next().await {
                if let tungstenite::Message::Text(cont) = cont {
                    if let Ok(it) = serde_json::from_str::<serde_json::Value>(&cont) {
                        let kind = it["type"].as_str();
                        let content = &it["content"];
    
                        if let Some(kind) = kind {
                            _ = incoming_sender.send((kind.to_string(), content.to_string()));
                        }
                    }
                } else if let tungstenite::Message::Ping(cont) = cont {
                    _ = outgoing_sender2.send(tungstenite::Message::Pong(cont));
                }
            }
        });

        WebsocketStream { incoming, outgoing_sender }
    }
}

#[async_trait]
impl Stream for WebsocketStream {
    type StreamError = tungstenite::error::Error;

    async fn stream_for<'a, T: serde::de::DeserializeOwned + Clone + ToOwned<Owned = T>>(&'a mut self, event: String) -> Box<dyn futures::stream::Stream<Item = std::borrow::Cow<T>> + 'a> {
        use futures::stream::StreamExt;

        let recv = self.incoming.clone();
        let recv = tokio_stream::wrappers::WatchStream::from(recv);

        Box::new(recv.flat_map(move |ev| {
            let (kind, data) = ev;
            if kind == event {
                if let Ok(it) = serde_json::from_str::<T>(&data) {
                    return futures::stream::iter(vec![std::borrow::Cow::Owned(it)]);
                }
            }
            return futures::stream::iter(vec![]);
        }))
    }

    async fn send<'a, T: serde::Serialize + Clone + ToOwned<Owned = T> + Sync>(&'a mut self, signal: String, item: &T) -> Result<(), StreamError<Self::StreamError>> {
        let inner = serde_json::to_value(item).map_err(StreamError::SerdeError)?;
        let json = serde_json::json!({
            "type": signal,
            "content": inner,
        });

        _ = self.outgoing_sender.send(tungstenite::Message::Text(json.to_string())).await;
        Ok(())
    }
}

pub struct HTTPSTransport {
    client: reqwest::Client,
    base_url: String,
}

pub enum HTTPSTransportError {
    ReqwestError(reqwest::Error),
    TungsteniteError(tungstenite::error::Error),
    HeaderError(reqwest::header::ToStrError),
}

impl From<HTTPSTransportError> for TransportError<HTTPSTransportError> {
    fn from(err: HTTPSTransportError) -> Self {
        Self::SelfError(err)
    }
}

#[async_trait]
impl Transport for HTTPSTransport {
    type Extra = reqwest::header::HeaderMap;
    type Stream = WebsocketStream;
    type TransportError = HTTPSTransportError;

    async fn make_request<
        In: serde::Serialize + std::marker::Send,
        Out: serde::de::DeserializeOwned,
        Error: serde::de::DeserializeOwned
    >(&mut self, endpoint: String, body: In, extra: Self::Extra) -> Result<Result<Out, Error>, TransportError<Self::TransportError>> {
        let body_str = serde_json::to_string(&body).map_err(TransportError::SerdeError)?;
        let response = self.client.post(self.base_url.clone() + &endpoint)
            .headers(extra)
            .body(body_str)
            .send()
            .await
            .map_err(HTTPSTransportError::ReqwestError)?;

        let status = response.status();
        let txt = response.text().await.map_err(HTTPSTransportError::ReqwestError)?;
        if status == 200 {
            let resp: Out = serde_json::from_str(&txt).map_err(TransportError::SerdeError)?;
            return Ok(Ok(resp));
        } else {
            let resp: Error = serde_json::from_str(&txt).map_err(TransportError::SerdeError)?;
            return Ok(Err(resp));
        }
    }
    async fn open_stream(&mut self, endpoint: String, extra: Self::Extra) -> Result<Self::Stream, TransportError<Self::TransportError>> {
        let mut it = serde_json::value::Map::new();
        
        for key in extra.keys() {
            it.insert(key.to_string(), serde_json::value::Value::String(extra[key].to_str().map_err(HTTPSTransportError::HeaderError)?.to_string()));
        }

        let path = self.base_url.clone() + &endpoint;
        let (mut socket, _) = tokio_tungstenite::connect_async(reqwest::Url::parse(&path).unwrap()).await.map_err(HTTPSTransportError::TungsteniteError)?;

        let body_str = serde_json::to_string(&it).map_err(TransportError::SerdeError)?;

        futures::SinkExt::send(&mut socket, tungstenite::Message::Text(body_str)).await.map_err(HTTPSTransportError::TungsteniteError)?;

        return Ok(WebsocketStream::init(socket));
    }
}

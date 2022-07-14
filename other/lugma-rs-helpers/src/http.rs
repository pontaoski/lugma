use super::{Stream, Transport};
use async_trait::async_trait;

struct WebsocketStream {
    stream: tokio_tungstenite::WebSocketStream<tokio_tungstenite::MaybeTlsStream<tokio::net::TcpStream>>
}

impl WebsocketStream {
    fn init(with: tokio_tungstenite::WebSocketStream<tokio_tungstenite::MaybeTlsStream<tokio::net::TcpStream>>) -> Self {
        WebsocketStream { stream: with }
    }
}

#[async_trait]
impl Stream for WebsocketStream {
    type StreamError = tungstenite::error::Error;

    async fn stream_for<T: serde::de::DeserializeOwned>(&mut self, event: String) -> std::pin::Pin<Box<dyn futures::stream::Stream<Item = T>>> {
        todo!()
    }

    async fn sink_for<T: serde::Serialize>(&mut self, signal: String) -> std::pin::Pin<Box<dyn futures::sink::Sink<T, Error = Self::StreamError>>> {
        todo!()
    }
}

struct HTTPSTransport {
    client: reqwest::Client,
    base_url: String,
}

#[async_trait]
impl Transport for HTTPSTransport {
    type Extra = reqwest::header::HeaderMap;
    type Stream = WebsocketStream;
    type TransportError = reqwest::Error;

    async fn make_request<
        In: serde::Serialize + std::marker::Send,
        Out: serde::de::DeserializeOwned,
        Error: serde::de::DeserializeOwned
    >(&mut self, endpoint: String, body: In, extra: Self::Extra) -> Result<Result<Out, Error>, Self::TransportError> {
        let body_str = serde_json::to_string(&body).unwrap();
        let response = self.client.post(self.base_url.clone() + &endpoint)
            .headers(extra)
            .body(body_str)
            .send()
            .await?;

        let status = response.status();
        let txt = response.text().await?;
        if status == 200 {
            let resp: Out = serde_json::from_str(&txt).unwrap();
            return Ok(Ok(resp));
        } else {
            let resp: Error = serde_json::from_str(&txt).unwrap();
            return Ok(Err(resp));
        }
    }
    async fn open_stream(&mut self, endpoint: String, extra: Self::Extra) -> Result<Self::Stream, <<Self as Transport>::Stream as Stream>::StreamError> {
        let mut it = serde_json::value::Map::new();
        
        for key in extra.keys() {
            it.insert(key.to_string(), serde_json::value::Value::String(extra[key].to_str().unwrap().to_string()));
        }

        let path = self.base_url.clone() + &endpoint;
        let (mut socket, _) = tokio_tungstenite::connect_async(reqwest::Url::parse(&path).unwrap()).await?;

        let body_str = serde_json::to_string(&it).unwrap();

        futures::SinkExt::send(&mut socket, tungstenite::Message::Text(body_str)).await?;

        return Ok(WebsocketStream::init(socket));
    }
}

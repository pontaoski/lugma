use async_trait::async_trait;

#[cfg(feature = "http_impl")]
pub mod http;

pub enum StreamError<T> {
    SelfError(T),
    SerdeError(serde_json::Error),
}

#[async_trait]
pub trait Stream {
    type StreamError;

    async fn stream_for<'a, T: serde::de::DeserializeOwned + Clone + ToOwned<Owned = T>>(&'a mut self, event: String) -> Box<dyn futures::stream::Stream<Item = std::borrow::Cow<T>> + 'a>;
    async fn send<'a, T: serde::Serialize + Clone + ToOwned<Owned = T> + Sync>(&'a mut self, signal: String, item: &T) -> Result<(), StreamError<Self::StreamError>>;
}

pub enum TransportError<T> {
    SelfError(T),
    SerdeError(serde_json::Error),
}

#[async_trait]
pub trait Transport {
    type Extra;
    type Stream: Stream;
    type TransportError;

    async fn make_request<
        In: serde::Serialize + std::marker::Send,
        Out: serde::de::DeserializeOwned,
        Error: serde::de::DeserializeOwned
    >(&mut self, endpoint: String, body: In, extra: Self::Extra) -> Result<Result<Out, Error>, TransportError<Self::TransportError>>;
    async fn open_stream(&mut self, endpoint: String, extra: Self::Extra) -> Result<Self::Stream, TransportError<Self::TransportError>>;
}

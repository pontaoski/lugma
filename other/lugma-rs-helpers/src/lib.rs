use serde;
use futures;
use async_trait::async_trait;

#[cfg(feature = "http_impl")]
pub mod http;

#[cfg(test)]
mod tests {
    #[test]
    fn it_works() {
        let result = 2 + 2;
        assert_eq!(result, 4);
    }
}

#[async_trait]
trait Stream {
    type StreamError;

    async fn stream_for<T: serde::de::DeserializeOwned>(&mut self, event: String) -> std::pin::Pin<Box<dyn futures::stream::Stream<Item = T>>>;
    async fn sink_for<T: serde::Serialize>(&mut self, signal: String) -> std::pin::Pin<Box<dyn futures::sink::Sink<T, Error = Self::StreamError>>>;
}

#[async_trait]
trait Transport {
    type Extra;
    type Stream: Stream;
    type TransportError;

    async fn make_request<
        In: serde::Serialize + std::marker::Send,
        Out: serde::de::DeserializeOwned,
        Error: serde::de::DeserializeOwned
    >(&mut self, endpoint: String, body: In, extra: Self::Extra) -> Result<Result<Out, Error>, Self::TransportError>;
    async fn open_stream(&mut self, endpoint: String, extra: Self::Extra) -> Result<Self::Stream, <<Self as Transport>::Stream as Stream>::StreamError>;
}

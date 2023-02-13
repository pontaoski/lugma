namespace Lugma.Client;

public interface ITransport
{
    Task<TReturn> MakeRequest<TRequest, TReturn, TError>(string endpoint, TRequest body, Metadata extra);
    Task<IStream> OpenStream(string endpoint, Metadata extra);
}

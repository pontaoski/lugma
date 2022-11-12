namespace Lugma;

public sealed class RPCError<TError> : Exception
{
    public TError Error { get; private set; }
    public RPCError(TError err)
    {
        Error = err;
    }
}
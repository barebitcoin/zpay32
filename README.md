This is a proof-of-concept for splitting out `zpay32` and `lnwire` from LND out into standalone packages. 

Major changes from upstream: 

*  Versioned handling of musig2v040 is gone
* `chainfee.SatPerKWeight` is replaced by a raw `btcutil.Amount`
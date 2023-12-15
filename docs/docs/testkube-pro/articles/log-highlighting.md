# Log Highlighting

export const ProBadge = () => {
  return (
    <span>
      <p class="pro-badge">PRO FEATURE</p>
    </span>
  );
}

<ProBadge />

## Overview

In Testkube Pro, we highlight relevant keywords in logs for faster debugging. To use this feature, open execution details.

On this screen, all the lines that may be relevant will be highlighted in the interface.

![log-highlighting.png](../../img/log-highlighting.png)

You may navigate through the highlighted lines with the arrows on top of the interface
or use the scrollbar where all relevant lines are marked.

## Filtering

To decide on the active highlight categories, you may click "Highlight for keywords" button.
By default, all the categories are active.

![log-highlighting-filtering.png](../../img/log-highlighting-filtering.png)

There are 4 categories at the moment, represented with few keywords each:

| Category                   | Keywords                                                            |
|----------------------------|---------------------------------------------------------------------|
| **Error Keywords**         | Error, Exception, Fail, Critical, Fatal                             |
| **Connection**             | Connection, Disconnect, Lost, Timeout, Refused, Handshake, Retrying |
| **Resource Issues**        | OutOfMemory, MemoryLeak, ResourceExhausted, LimitExceeded, Quota    |
| **Access & Authorization** | Denied, Unauthorized, Forbidden, Invalid, Invalid Token, Expired    |
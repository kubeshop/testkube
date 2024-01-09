# Deprecations

Software deprecation refers to the process of phasing out or discontinuing support for a particular software feature. This decision is typically made by software developers or vendors due to various reasons such as security concerns, outdated technology, or the introduction of more efficient alternatives.

Usually if possible and reasonable we try to keep the backward compatibility.

## List of Testkube deprecations

### Since `v1.16.16` internal `/results` route 

Reason of deprecation was that after Fiber (https://gofiber.io/) security update disallowing of using `Mount` in simple way. 
Also route is not used by the Testkube internally anymore.

As a workaround users who want's to configure own ingresses should use `/` route instead.

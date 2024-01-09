# Deprecations

Software deprecation refers to the process of phasing out or discontinuing support for a particular software feature, API (Application Programming Interface), or an entire software product. This decision is typically made by software developers or vendors due to various reasons such as security concerns, outdated technology, or the introduction of more efficient alternatives.

Some of deprecations in Testkube are caused by security reasons, some of them are caused by changes in dependencies. Usually if possible and reasonable we try to keep backward compatibility.

## List of Testkube deprecations

### Since `v1.16.16` internal `/results` route 

Reason of deprecation was that after Fiber (https://gofiber.io/) security update disallowing of using `Mount` in simple way. 
Also route is not used by Testkube internally anymore

As a workaround users who want's to configure own ingresses should use `/` route instead.

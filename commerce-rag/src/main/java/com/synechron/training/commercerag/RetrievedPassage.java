package com.synechron.training.commercerag;

import java.util.Objects;

public record RetrievedPassage(
        String passageId,
        String scopeId,
        String content,
        String sourceId,
        String sourceTitle,
        String sourceUri) {

    public RetrievedPassage {
        Objects.requireNonNull(passageId, "passageId must not be null");
        Objects.requireNonNull(scopeId, "scopeId must not be null");
        Objects.requireNonNull(content, "content must not be null");
        Objects.requireNonNull(sourceId, "sourceId must not be null");
        Objects.requireNonNull(sourceTitle, "sourceTitle must not be null");
        Objects.requireNonNull(sourceUri, "sourceUri must not be null");
    }
}
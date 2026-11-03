package com.synechron.training.commercerag;

import java.util.Objects;

public record Citation(String passageId, String sourceId, String title, String uri) {
    public Citation {
        Objects.requireNonNull(passageId, "passageId must not be null");
        Objects.requireNonNull(sourceId, "sourceId must not be null");
        Objects.requireNonNull(title, "title must not be null");
        Objects.requireNonNull(uri, "uri must not be null");
    }
}
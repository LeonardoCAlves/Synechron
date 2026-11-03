package com.synechron.training.commercerag;

import java.util.List;
import java.util.Objects;

public record Answer(String text, List<Citation> citations, boolean abstained) {
    public Answer {
        Objects.requireNonNull(text, "text must not be null");
        citations = List.copyOf(citations);
    }
}
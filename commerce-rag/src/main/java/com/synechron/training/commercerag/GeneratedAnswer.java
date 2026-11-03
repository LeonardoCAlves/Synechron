package com.synechron.training.commercerag;

import java.util.List;
import java.util.Objects;

public record GeneratedAnswer(String text, List<String> citedPassageIds) {
    public GeneratedAnswer {
        Objects.requireNonNull(text, "text must not be null");
        citedPassageIds = List.copyOf(citedPassageIds);
    }
}
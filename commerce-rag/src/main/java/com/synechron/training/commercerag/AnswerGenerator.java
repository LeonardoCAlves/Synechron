package com.synechron.training.commercerag;

import java.util.List;

public interface AnswerGenerator {
    GeneratedAnswer generate(String question, List<RetrievedPassage> passages);
}
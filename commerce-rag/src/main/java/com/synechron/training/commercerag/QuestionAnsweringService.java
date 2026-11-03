package com.synechron.training.commercerag;

import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Objects;

public final class QuestionAnsweringService {
    public static final String NO_EVIDENCE_MESSAGE =
            "I could not find reliable information to answer that question.";

    private final Retriever retriever;
    private final AnswerGenerator generator;
    private final int maxPassages;

    public QuestionAnsweringService(
            Retriever retriever,
            AnswerGenerator generator,
            int maxPassages) {
        this.retriever = Objects.requireNonNull(retriever, "retriever must not be null");
        this.generator = Objects.requireNonNull(generator, "generator must not be null");
        if (maxPassages < 1) {
            throw new IllegalArgumentException("maxPassages must be positive");
        }
        this.maxPassages = maxPassages;
    }

    public Answer answer(String question, String scopeId) {
        String normalizedQuestion = requireNonBlank(question, "question");
        String normalizedScopeId = requireNonBlank(scopeId, "scopeId");

        Map<String, RetrievedPassage> passagesById = new LinkedHashMap<>();
        for (RetrievedPassage passage : retriever.retrieve(
                normalizedQuestion,
                normalizedScopeId,
                maxPassages)) {
            if (!passage.scopeId().equals(normalizedScopeId)
                    || passagesById.containsKey(passage.passageId())) {
                continue;
            }
            passagesById.put(passage.passageId(), passage);
            if (passagesById.size() == maxPassages) {
                break;
            }
        }

        if (passagesById.isEmpty()) {
            return abstain();
        }

        List<RetrievedPassage> passages = List.copyOf(passagesById.values());
        GeneratedAnswer generated = generator.generate(normalizedQuestion, passages);
        List<Citation> citations = validatedCitations(
                generated.citedPassageIds(),
                passagesById);
        String answerText = generated.text().strip();
        if (answerText.isEmpty() || citations.isEmpty()) {
            return abstain();
        }

        return new Answer(answerText, citations, false);
    }

    private static List<Citation> validatedCitations(
            List<String> citedPassageIds,
            Map<String, RetrievedPassage> passagesById) {
        List<Citation> citations = new ArrayList<>();
        LinkedHashSet<String> seenPassageIds = new LinkedHashSet<>();
        for (String passageId : citedPassageIds) {
            RetrievedPassage passage = passagesById.get(passageId);
            if (passage == null || !seenPassageIds.add(passageId)) {
                continue;
            }
            citations.add(new Citation(
                    passage.passageId(),
                    passage.sourceId(),
                    passage.sourceTitle(),
                    passage.sourceUri()));
        }
        return List.copyOf(citations);
    }

    private static String requireNonBlank(String value, String fieldName) {
        if (value == null || value.isBlank()) {
            throw new IllegalArgumentException(fieldName + " must not be blank");
        }
        return value.strip();
    }

    private static Answer abstain() {
        return new Answer(NO_EVIDENCE_MESSAGE, List.of(), true);
    }
}
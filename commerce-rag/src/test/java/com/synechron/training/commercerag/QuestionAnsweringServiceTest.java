package com.synechron.training.commercerag;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.ArrayList;
import java.util.List;
import org.junit.jupiter.api.Test;

class QuestionAnsweringServiceTest {
    @Test
    void scopesContextAndOnlyReturnsRetrievedCitations() {
        FakeRetriever retriever = new FakeRetriever(List.of(
                passage("foreign", "store-2"),
                passage("first", "store-1"),
                passage("second", "store-1"),
                passage("third", "store-1")));
        FakeGenerator generator = new FakeGenerator(new GeneratedAnswer(
                "This answer is supported.",
                List.of("first", "invented", "first", "second")));
        QuestionAnsweringService service = new QuestionAnsweringService(
                retriever,
                generator,
                2);

        Answer answer = service.answer("  return policy?  ", " store-1 ");

        assertEquals("return policy?", retriever.query);
        assertEquals("store-1", retriever.scopeId);
        assertEquals(2, retriever.limit);
        assertEquals(List.of("first", "second"),
                generator.passages.stream().map(RetrievedPassage::passageId).toList());
        assertEquals(List.of("first", "second"),
                answer.citations().stream().map(Citation::passageId).toList());
        assertFalse(answer.abstained());
    }

    @Test
    void abstainsWithoutEvidenceAndSkipsGeneration() {
        FakeRetriever retriever = new FakeRetriever(List.of());
        FakeGenerator generator = new FakeGenerator(new GeneratedAnswer("Unused", List.of()));
        QuestionAnsweringService service = new QuestionAnsweringService(retriever, generator, 5);

        Answer answer = service.answer("Where is my order?", "store-1");

        assertEquals(QuestionAnsweringService.NO_EVIDENCE_MESSAGE, answer.text());
        assertTrue(answer.abstained());
        assertTrue(generator.passages.isEmpty());
    }

    @Test
    void abstainsWhenGeneratorCitesNoRetrievedEvidence() {
        FakeRetriever retriever = new FakeRetriever(List.of(passage("known", "store-1")));
        FakeGenerator generator = new FakeGenerator(new GeneratedAnswer(
                "An unsupported answer.",
                List.of("not-retrieved")));
        QuestionAnsweringService service = new QuestionAnsweringService(retriever, generator, 5);

        Answer answer = service.answer("Question", "store-1");

        assertEquals(QuestionAnsweringService.NO_EVIDENCE_MESSAGE, answer.text());
        assertEquals(List.of(), answer.citations());
        assertTrue(answer.abstained());
    }

    @Test
    void rejectsBlankQuestionOrScope() {
        QuestionAnsweringService service = new QuestionAnsweringService(
                new FakeRetriever(List.of()),
                new FakeGenerator(new GeneratedAnswer("Unused", List.of())),
                5);

        assertThrows(IllegalArgumentException.class, () -> service.answer("  ", "store-1"));
        assertThrows(IllegalArgumentException.class, () -> service.answer("Question", "  "));
    }

    @Test
    void requiresPositivePassageLimit() {
        assertThrows(IllegalArgumentException.class, () -> new QuestionAnsweringService(
                new FakeRetriever(List.of()),
                new FakeGenerator(new GeneratedAnswer("Unused", List.of())),
                0));
    }

    private static RetrievedPassage passage(String id, String scopeId) {
        return new RetrievedPassage(
                id,
                scopeId,
                "Evidence for " + id,
                "source-" + id,
                "Title " + id,
                "https://example.test/" + id);
    }

    private static final class FakeRetriever implements Retriever {
        private final List<RetrievedPassage> passages;
        private String query;
        private String scopeId;
        private int limit;

        private FakeRetriever(List<RetrievedPassage> passages) {
            this.passages = List.copyOf(passages);
        }

        @Override
        public List<RetrievedPassage> retrieve(String query, String scopeId, int limit) {
            this.query = query;
            this.scopeId = scopeId;
            this.limit = limit;
            return passages;
        }
    }

    private static final class FakeGenerator implements AnswerGenerator {
        private final GeneratedAnswer generatedAnswer;
        private List<RetrievedPassage> passages = new ArrayList<>();

        private FakeGenerator(GeneratedAnswer generatedAnswer) {
            this.generatedAnswer = generatedAnswer;
        }

        @Override
        public GeneratedAnswer generate(String question, List<RetrievedPassage> passages) {
            this.passages = List.copyOf(passages);
            return generatedAnswer;
        }
    }
}
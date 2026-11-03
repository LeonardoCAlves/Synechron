package com.synechron.training.commercerag;

import java.util.List;

public interface Retriever {
    List<RetrievedPassage> retrieve(String query, String scopeId, int limit);
}
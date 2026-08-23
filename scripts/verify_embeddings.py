import os
import sys
import numpy as np
from tokenizers import Tokenizer
import onnxruntime as ort

def mean_pooling(token_embeddings: np.ndarray, attention_mask: np.ndarray) -> np.ndarray:
    """Mean pooling taking into account the attention mask (matches Go meanPool3D)."""
    input_mask_expanded = np.expand_dims(attention_mask, axis=-1).astype(np.float32)
    sum_embeddings = np.sum(token_embeddings * input_mask_expanded, axis=1)
    sum_mask = np.clip(input_mask_expanded.sum(axis=1), a_min=1e-9, a_max=None)
    return sum_embeddings / sum_mask

def normalize_l2(vector: np.ndarray) -> np.ndarray:
    """L2 normalize vector (matches Go normalizeL2)."""
    norm = np.linalg.norm(vector)
    if norm == 0:
        return vector
    return vector / norm

def test_embedding(text: str, tokenizer_path: str, model_path: str):
    print("=" * 60)
    print(f"Testing text: \"{text}\"")
    print("=" * 60)

    # 1. Load Tokenizer
    tok = Tokenizer.from_file(tokenizer_path)
    encoded = tok.encode(text)

    input_ids = np.array([encoded.ids], dtype=np.int64)
    attention_mask = np.array([encoded.attention_mask], dtype=np.int64)
    token_type_ids = np.array([encoded.type_ids], dtype=np.int64)

    print(f"Token count: {len(encoded.ids)}")
    print(f"Tokens: {encoded.tokens[:15]}{'...' if len(encoded.tokens) > 15 else ''}")
    print(f"Input IDs (first 10): {encoded.ids[:10]}")
    print(f"Attention mask (first 10): {encoded.attention_mask[:10]}")

    # 2. Run ONNX Session
    session = ort.InferenceSession(model_path)
    
    # Map inputs dynamically
    feed_dict = {}
    for inp in session.get_inputs():
        name_lower = inp.name.lower()
        if "attention" in name_lower:
            feed_dict[inp.name] = attention_mask
        elif "token_type" in name_lower or "segment" in name_lower:
            feed_dict[inp.name] = token_type_ids
        else:
            feed_dict[inp.name] = input_ids

    outputs = session.run(None, feed_dict)
    raw_output = outputs[0]  # Expected shape: (1, seq_len, hidden_dim) or (1, hidden_dim)
    print(f"Raw ONNX output shape: {raw_output.shape}")

    # 3. Pooling
    if len(raw_output.shape) == 3:
        pooled = mean_pooling(raw_output, attention_mask)[0]
    elif len(raw_output.shape) == 2:
        pooled = raw_output[0]
    else:
        pooled = raw_output.flatten()

    # 4. L2 Normalization
    normed = normalize_l2(pooled)

    print(f"Embedding dimension: {len(normed)}")
    print(f"Vector L2 Norm: {np.linalg.norm(normed):.6f}")
    print(f"First 10 vector values: {[round(float(x), 6) for x in normed[:10]]}")
    return normed

def main():
    root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
    tokenizer_path = os.path.join(root_dir, "asset", "tokenizer.json")
    model_path = os.path.join(root_dir, "asset", "model_int8.onnx")

    if not os.path.exists(tokenizer_path):
        print(f"Error: Tokenizer not found at {tokenizer_path}")
        sys.exit(1)
    if not os.path.exists(model_path):
        print(f"Error: Model not found at {model_path}")
        sys.exit(1)

    print(f"Loaded assets from:\n  Tokenizer: {tokenizer_path}\n  Model: {model_path}\n")

    # Run tests on sample texts
    v1 = test_embedding("search_query: What is cellular respiration?", tokenizer_path, model_path)
    v2 = test_embedding("search_document: Cellular respiration is a set of metabolic reactions that take place in the cells.", tokenizer_path, model_path)
    v3 = test_embedding("search_document: The Eiffel Tower is a wrought-iron lattice tower on the Champ de Mars in Paris.", tokenizer_path, model_path)

    # Cosine similarities
    sim_related = float(np.dot(v1, v2))
    sim_unrelated = float(np.dot(v1, v3))

    print("\n" + "=" * 60)
    print("Semantic Similarity Verification:")
    print(f"  Similarity(Query, Biology Chunk):    {sim_related:.4f} (Expected high > 0.8)")
    print(f"  Similarity(Query, Unrelated Chunk):  {sim_unrelated:.4f} (Expected low < 0.5)")
    print("=" * 60)

    # Export reference fixture for Go unit test verification
    fixture_path = os.path.join(root_dir, "internal", "embeddings", "testdata", "reference_vector.json")
    os.makedirs(os.path.dirname(fixture_path), exist_ok=True)
    with open(fixture_path, "w") as f:
        json.dump({
            "text": "search_query: What is cellular respiration?",
            "embedding": v1.tolist(),
            "first_10": [float(x) for x in v1[:10]]
        }, f, indent=2)
    print(f"\n[OK] Wrote reference fixture to: {fixture_path}")

if __name__ == "__main__":
    import json
    main()


use serde::{Deserialize, Serialize};

/// Basit keyword + pattern tabanlı Spam skoru hesaplama motoru.
/// Gerçek NLP/ML entegrasyonu için Phase 2'de PyTorch veya ONNX Runtime eklenebilir.
pub struct AntiSpamEngine;

impl AntiSpamEngine {
    /// E-posta içeriğini analiz eder ve spam skoru döndürür (0.0 - 1.0)
    pub fn analyze(subject: &str, body: &str, sender: &str) -> f64 {
        let mut score: f64 = 0.0;

        // Kural 1: Kara liste kelimeleri
        let blacklist_words = [
            "viagra", "casino", "lottery", "winner", "free money",
            "click here", "unsubscribe", "act now", "limited time",
            "nigerian prince", "bitcoin profit", "crypto invest",
        ];

        let combined = format!("{} {} {}", subject, body, sender).to_lowercase();

        for word in &blacklist_words {
            if combined.contains(word) {
                score += 0.15;
            }
        }

        // Kural 2: Çok fazla büyük harf kullanımı
        let uppercase_ratio = body.chars().filter(|c| c.is_uppercase()).count() as f64
            / body.len().max(1) as f64;
        if uppercase_ratio > 0.5 {
            score += 0.2;
        }

        // Kural 3: Çok fazla link içermesi
        let link_count = body.matches("http").count();
        if link_count > 5 {
            score += 0.15;
        }

        // Kural 4: Bilinmeyen TLD'ler
        let suspicious_tlds = [".xyz", ".top", ".click", ".buzz", ".gq", ".tk"];
        for tld in &suspicious_tlds {
            if sender.ends_with(tld) {
                score += 0.25;
            }
        }

        // Skor 1.0'ı aşmasın
        score.min(1.0)
    }

    /// Skor eşik değerine göre karar verir
    pub fn is_spam(subject: &str, body: &str, sender: &str, threshold: f64) -> bool {
        Self::analyze(subject, body, sender) >= threshold
    }
}

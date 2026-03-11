# SmartPath FX: High-Precision Currency Routing Engine

**SmartPath FX** is a specialized backend engine built in **Pure Go** designed to identify the most cost-effective routes for international money transfers. By leveraging **Dijkstra’s Shortest Path Algorithm**, the system discovers "Smart Routes" through bridge currencies (e.g., USD, GBP) that often outperform direct exchange pairs in terms of total landing cost.

## The Problem
Standard foreign exchange (FX) providers often have higher spreads on direct "exotic" pairs (e.g., BRL to AOA). However, by performing a triangular conversion (BRL → GBP → AOA), users can sometimes achieve a higher final amount, even after accounting for multiple fixed fees.

## The Solution
This engine treats the currency market as a **Directed Weighted Graph**:
* **Nodes**: Currencies (USD, EUR, GBP, AOA, BRL, etc.).
* **Edges**: Exchange rates between currencies.
* **Weights**: Calculated based on `(Amount - FixedFee) * Rate` to maximize the final output.

### Key Features
* **Zero Dependencies**: Built using only the Go Standard Library for maximum performance and security.
* **Smart Routing**: Implements a customized Dijkstra algorithm focused on profit maximization rather than just distance.
* **Fee-Aware Logic**: Normalizes fixed fees across different currencies to ensure mathematical integrity.
* **Concurrency**: Uses Goroutines and Channels for high-speed API polling and route calculation.
* **Production-Ready API**: Returns detailed summaries including direct vs. smart comparisons and total savings.

## How It Works
1.  **Data Ingestion**: The engine fetches real-time rates from providers (currently integrated with **Wise**).
2.  **Graph Construction**: It maps all possible connections, including intermediate "bridge" nodes.
3.  **Pathfinding**: Dijkstra’s algorithm traverses the graph to find the sequence of jumps that yields the highest `final_amount`.
4.  **Validation**: The system compares the optimized path against the direct path to calculate real savings.

## Tech Stack
* **Language**: Go 1.2x (Pure Go)
* **Architecture**: Clean Architecture / Domain-Driven Design
* **API**: RESTful JSON API

## Official API Response
```json
{
    "request": {
        "from": "EUR",
        "to": "BRL",
        "amount": 10000000
    },
    "summary": {
        "smart_final_amount": 59974918.08,
        "direct_final_amount": 59974164.02,
        "total_savings": 754.06,
        "savings_percentage": 0.0013,
        "total_fixed_fees_source_currency": 12.94
    },
    "smart_path": [
        {
            "from": "EUR",
            "to": "GBP",
            "value": 0.86475,
            "fixed_fee": 6,
            "fee_currency": "EUR",
            "provider": "Wise",
            "last_update": "2026-03-11T13:53:05.756389+05:30"
        },
        {
            "from": "GBP",
            "to": "BRL",
            "value": 6.93553,
            "fixed_fee": 6,
            "fee_currency": "GBP",
            "provider": "Wise",
            "last_update": "2026-03-11T13:53:05.74836+05:30"
        }
    ],
    "meta": {
        "confidence_score": 100,
        "timestamp": "2026-03-11T13:53:05.787179+05:30",
        "efficiency": "Standard"
    }
}
```
## Roadmap & Future Work
* []Multi-Provider Support: Expand graph edges by integrating Revolut and Western Union APIs
* []Precision Upgrade: Transition from floating-point to fixed-point decimal libraries for financial-grade math

## Contributing
Contributions make the open-source community an amazing place to learn and create
1.  Fork the Project
2.  Create your Feature Branch (git checkout -b feature/SmartRoute)
3.  Commit your Changes (git commit -m 'Add some SmartRoute')
4.  Push to the Branch (git push origin feature/SmartRoute)
5.  Open a Pull Request

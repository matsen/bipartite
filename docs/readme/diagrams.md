# README Diagrams

## Diagram 1: My Digital World (3-spoke)

Simple hub-and-spoke showing where a team lead's time goes.

```mermaid
graph TD
    ME((me))
    SLACK[Slack<br/>8,077 messages]
    GH[GitHub<br/>1,607 commits]
    PDF[Papers<br/>100s evaluated]

    ME --- SLACK
    ME --- GH
    ME --- PDF

    style ME fill:#4a90d9,stroke:#333,color:#fff
    style SLACK fill:#4a154b,stroke:#333,color:#fff
    style GH fill:#24292e,stroke:#333,color:#fff
    style PDF fill:#d9534f,stroke:#333,color:#fff
```

### Alternate version (horizontal)

```mermaid
graph LR
    SLACK[Slack<br/>8,077 messages]
    ME((me))
    PDF[Papers<br/>100s evaluated]
    GH[GitHub<br/>1,607 commits]

    SLACK --- ME
    ME --- GH
    ME --- PDF

    style ME fill:#4a90d9,stroke:#333,color:#fff
    style SLACK fill:#4a154b,stroke:#333,color:#fff
    style GH fill:#24292e,stroke:#333,color:#fff
    style PDF fill:#d9534f,stroke:#333,color:#fff
```

## Diagram 2: Knowledge Graph (Inside/Outside)

The bipartite structure: internal world on left, external literature on right.

```mermaid
graph LR
    subgraph internal [" Your World "]
        direction TB
        R1[repo: netam]
        R2[repo: epiphyte]
        C1([concept: variational inference])
        C2([concept: phylogenetics])
        P1[project: DASM]
    end

    subgraph external [" Literature "]
        direction TB
        L1[Kingma 2014]
        L2[Felsenstein 1981]
        L3[Bishop 2006]
        L4[Yang 2006]
    end

    R1 -.->|uses| C1
    R2 -.->|uses| C2
    P1 -.->|encompasses| R1
    P1 -.->|encompasses| R2

    C1 ===|introduced by| L1
    C1 ===|explained in| L3
    C2 ===|founded by| L2
    C2 ===|textbook| L4

    style internal fill:#e8f4e8,stroke:#2d5a2d
    style external fill:#e8e8f4,stroke:#2d2d5a
```

### Alternate: Vertical divide

```mermaid
graph TB
    subgraph internal [" Internal: Your Group "]
        direction LR
        R1[netam]
        R2[epiphyte]
        C1([VI])
        C2([phylo])
        P1[DASM]

        P1 --> R1
        P1 --> R2
        R1 -.-> C1
        R2 -.-> C2
    end

    subgraph external [" External: Literature "]
        direction LR
        L1[Kingma 2014]
        L2[Felsenstein 1981]
        L3[Bishop 2006]
    end

    C1 ==> L1
    C1 ==> L3
    C2 ==> L2

    style internal fill:#f0fff0,stroke:#228b22
    style external fill:#f0f0ff,stroke:#4169e1
```

---

## Stats to highlight

From the user's 2025 data, the most impressive combo might be:

- **8,077 Slack messages** - shows communication load
- **1,607 commits** - shows active IC work too
- **186 PRs reviewed** - shows leadership/mentorship role

Or pick two:
- "1,600+ commits and 186 PR reviews" (emphasizes dual role)
- "8,000+ Slack messages evaluating 100s of papers" (emphasizes coordination)

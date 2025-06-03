# Iddaa Events API Documentation

## Events Endpoint

**URL:** `GET https://sportsbookv2.iddaa.com/sportsbook/events?st=1&type=0&version=0`

**Response Format:**

```json
{
   "isSuccess": true,
   "data": {
      "isdiff": false,
      "version": 1748922170004,
      "events": [
         {
            "i": 2186060,
            "bri": 60832245,
            "v": 500182060,
            "hn": "Millonarios",
            "an": "Atl Nacional",
            "sid": 1,
            "s": 0,
            "bp": 0,
            "il": true,
            "mbc": 3,
            "kOdd": true,
            "m": [
               {
                  "i": 40948298,
                  "t": 2,
                  "st": 60,
                  "v": 499965554,
                  "s": 1,
                  "mbc": 3,
                  "sov": "0.5",
                  "o": [
                     {
                        "no": 1,
                        "odd": 2.32,
                        "wodd": 2.22,
                        "n": "Alt"
                     },
                     {
                        "no": 2,
                        "odd": 1.36,
                        "wodd": 1.30,
                        "n": "Üst"
                     }
                  ]
               }
            ],
            "ci": 1242,
            "oc": 40,
            "d": 1749172500,
            "hc": false
         }
      ]
   }
}
```

## Field Meanings

### Root Response
- `isSuccess`: API call success status
- `data`: Response data wrapper
- `message`: Error message (if any)

### Data Object
- `isdiff`: Whether this is a differential update
- `version`: API version timestamp
- `events`: Array of event objects

### Event Object
- `i`: Event ID (unique identifier)
- `bri`: Bulletin Reference ID
- `v`: Version number
- `hn`: Home team name
- `an`: Away team name
- `sid`: Sport ID (1=Football, 2=Basketball, etc.)
- `s`: Status (0=Scheduled, 1=Live, 2=Finished, etc.)
- `bp`: Bet Program ID
- `il`: Is Live flag
- `mbc`: Market Base Count
- `kOdd`: Has King Odd flag
- `m`: Markets array
- `ci`: Competition ID
- `oc`: Odds Count
- `d`: Event date (Unix timestamp in milliseconds)
- `hc`: Has Combine flag

### Market Object (`m` array)
- `i`: Market ID
- `t`: Market type (1=Main, 2=Alternative)
- `st`: Sub-type (betting market code)
  - `1`: 1X2 (Match Result)
  - `60`: Over/Under 0.5 Goals
  - `101`: Over/Under 2.5 Goals
  - `89`: Both Teams to Score
  - `88`: Half Time Result
  - `92`: Double Chance
  - `77`: Draw No Bet
  - `91`: Total Goals Odd/Even
  - `720`: Red Card
  - `36`: Exact Score
  - `603`: Home Team Over/Under Goals
  - `604`: Away Team Over/Under Goals
  - `722`: Home Team Corner Kick
  - `723`: Away Team Corner Kick
- `v`: Version number
- `s`: Status
- `mbc`: Market Base Count
- `sov`: Special Outcome Value (e.g., "0.5", "2.5" for Over/Under)
- `o`: Outcomes array

### Outcome Object (`o` array)
- `no`: Outcome number/position
- `odd`: Current odds value
- `wodd`: Winning odds value (adjusted)
- `n`: Outcome name (in Turkish)
  - `"1"`: Home team win
  - `"0"`: Draw
  - `"2"`: Away team win
  - `"Alt"`: Under
  - `"Üst"`: Over
  - `"Var"`: Yes
  - `"Yok"`: No
  - `"Tek"`: Odd
  - `"Çift"`: Even

## Market Sub-types Reference

| Code | Market Type | Description |
|------|-------------|-------------|
| 1    | 1X2         | Match Result (Home/Draw/Away) |
| 60   | O/U 0.5     | Over/Under 0.5 Goals |
| 101  | O/U 2.5     | Over/Under 2.5 Goals |
| 89   | BTTS        | Both Teams to Score |
| 88   | HT          | Half Time Result |
| 92   | DC          | Double Chance |
| 77   | DNB         | Draw No Bet |
| 91   | O/E         | Total Goals Odd/Even |
| 720  | Red Card    | Red Card in Match |
| 36   | Exact Score | Exact Final Score |
| 603  | Home O/U    | Home Team Over/Under Goals |
| 604  | Away O/U    | Away Team Over/Under Goals |
| 722  | Home Corner | Home Team Corner Kicks |
| 723  | Away Corner | Away Team Corner Kicks |

## Turkish Outcome Names Translation

| Turkish | English |
|---------|---------|
| Alt     | Under   |
| Üst     | Over    |
| Var     | Yes     |
| Yok     | No      |
| Tek     | Odd     |
| Çift    | Even    |
| Evet    | Yes     |
| Hayır   | No      |
| Eşit    | Equal   |

## Status Codes

### Event Status (`s`)
- `0`: Scheduled
- `1`: Live
- `2`: Finished
- `3`: Postponed
- `4`: Cancelled

### Market Status (`s`)
- `0`: Inactive
- `1`: Active
- `2`: Suspended
- `3`: Settled

## Sport IDs (`sid`)
- `1`: Football (Soccer)
- `2`: Basketball
- `4`: Ice Hockey
- `5`: Tennis
- `6`: Handball
- `11`: Formula 1
- `23`: Other Sports
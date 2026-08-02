import type { Mentality, PlayerPosition } from './types'

export type PremierLeaguePlayerSeed = {
  name: string
  position: PlayerPosition
  age: number
  ability: number
  form: number
}

export type PremierLeagueClubSeed = {
  id: string
  name: string
  shortName: string
  city: string
  badgeUrl: string
  colors: [string, string]
  strength: number
  style: Mentality
  players: PremierLeaguePlayerSeed[]
}

export const PREMIER_LEAGUE_DATA_SNAPSHOT = '2026-08-02'

export const premierLeagueClubSeeds: PremierLeagueClubSeed[] = [
  {
    "id": "arsenal",
    "name": "Arsenal",
    "shortName": "ARS",
    "city": "Arsenal",
    "badgeUrl": "https://media.api-sports.io/football/teams/42.png",
    "colors": [
      "#db0007",
      "#ffffff"
    ],
    "strength": 86,
    "style": "Assertive",
    "players": [
      {
        "name": "David Raya Martín",
        "position": "GK",
        "age": 30,
        "ability": 79,
        "form": 79
      },
      {
        "name": "Gabriel dos Santos Magalhães",
        "position": "DEF",
        "age": 28,
        "ability": 87,
        "form": 91
      },
      {
        "name": "Jurriën Timber",
        "position": "DEF",
        "age": 25,
        "ability": 81,
        "form": 81
      },
      {
        "name": "William Saliba",
        "position": "DEF",
        "age": 25,
        "ability": 79,
        "form": 78
      },
      {
        "name": "Riccardo Calafiori",
        "position": "DEF",
        "age": 24,
        "ability": 77,
        "form": 74
      },
      {
        "name": "Declan Rice",
        "position": "MID",
        "age": 27,
        "ability": 84,
        "form": 83
      },
      {
        "name": "Bukayo Saka",
        "position": "MID",
        "age": 24,
        "ability": 87,
        "form": 81
      },
      {
        "name": "Martín Zubimendi Ibáñez",
        "position": "MID",
        "age": 27,
        "ability": 77,
        "form": 73
      },
      {
        "name": "Eberechi Eze",
        "position": "MID",
        "age": 28,
        "ability": 79,
        "form": 70
      },
      {
        "name": "Martin Ødegaard",
        "position": "MID",
        "age": 27,
        "ability": 78,
        "form": 67
      },
      {
        "name": "Viktor Gyökeres",
        "position": "FWD",
        "age": 28,
        "ability": 81,
        "form": 72
      },
      {
        "name": "Kai Havertz",
        "position": "FWD",
        "age": 27,
        "ability": 80,
        "form": 64
      },
      {
        "name": "Piero Hincapié",
        "position": "DEF",
        "age": 24,
        "ability": 76,
        "form": 70
      },
      {
        "name": "Noni Madueke",
        "position": "MID",
        "age": 24,
        "ability": 77,
        "form": 65
      },
      {
        "name": "Mikel Merino Zazón",
        "position": "MID",
        "age": 30,
        "ability": 76,
        "form": 65
      },
      {
        "name": "Gabriel Martinelli Silva",
        "position": "MID",
        "age": 25,
        "ability": 76,
        "form": 59
      },
      {
        "name": "Benjamin White",
        "position": "DEF",
        "age": 28,
        "ability": 76,
        "form": 69
      },
      {
        "name": "Cristhian Mosquera",
        "position": "DEF",
        "age": 22,
        "ability": 74,
        "form": 59
      }
    ]
  },
  {
    "id": "aston-villa",
    "name": "Aston Villa",
    "shortName": "AVL",
    "city": "Aston Villa",
    "badgeUrl": "https://media.api-sports.io/football/teams/66.png",
    "colors": [
      "#670e36",
      "#95bfe5"
    ],
    "strength": 79,
    "style": "Measured",
    "players": [
      {
        "name": "Emiliano Martínez Romero",
        "position": "GK",
        "age": 33,
        "ability": 76,
        "form": 75
      },
      {
        "name": "Matty Cash",
        "position": "DEF",
        "age": 28,
        "ability": 74,
        "form": 72
      },
      {
        "name": "Ezri Konsa Ngoyo",
        "position": "DEF",
        "age": 28,
        "ability": 73,
        "form": 70
      },
      {
        "name": "Lucas Digne",
        "position": "DEF",
        "age": 33,
        "ability": 73,
        "form": 68
      },
      {
        "name": "Pau Torres",
        "position": "DEF",
        "age": 29,
        "ability": 73,
        "form": 67
      },
      {
        "name": "John McGinn",
        "position": "MID",
        "age": 31,
        "ability": 76,
        "form": 71
      },
      {
        "name": "Emiliano Buendía Stati",
        "position": "MID",
        "age": 29,
        "ability": 76,
        "form": 66
      },
      {
        "name": "Joao Gomes",
        "position": "MID",
        "age": 25,
        "ability": 76,
        "form": 68
      },
      {
        "name": "Amadou Onana",
        "position": "MID",
        "age": 24,
        "ability": 74,
        "form": 67
      },
      {
        "name": "Boubacar Kamara",
        "position": "MID",
        "age": 26,
        "ability": 75,
        "form": 69
      },
      {
        "name": "Ollie Watkins",
        "position": "FWD",
        "age": 30,
        "ability": 84,
        "form": 79
      },
      {
        "name": "Tammy Abraham",
        "position": "FWD",
        "age": 28,
        "ability": 73,
        "form": 57
      },
      {
        "name": "Alejandro Garnacho Ferreyra",
        "position": "MID",
        "age": 22,
        "ability": 76,
        "form": 62
      },
      {
        "name": "Ross Barkley",
        "position": "MID",
        "age": 32,
        "ability": 73,
        "form": 63
      },
      {
        "name": "Ian Maatsen",
        "position": "DEF",
        "age": 24,
        "ability": 72,
        "form": 60
      },
      {
        "name": "Tyrone Mings",
        "position": "DEF",
        "age": 33,
        "ability": 73,
        "form": 66
      },
      {
        "name": "Evann Guessand",
        "position": "MID",
        "age": 25,
        "ability": 74,
        "form": 60
      },
      {
        "name": "Lamare Bogarde",
        "position": "MID",
        "age": 22,
        "ability": 72,
        "form": 56
      }
    ]
  },
  {
    "id": "bournemouth",
    "name": "Bournemouth",
    "shortName": "BOU",
    "city": "Bournemouth",
    "badgeUrl": "https://media.api-sports.io/football/teams/35.png",
    "colors": [
      "#da291c",
      "#000000"
    ],
    "strength": 73,
    "style": "Assertive",
    "players": [
      {
        "name": "Đorđe Petrović",
        "position": "GK",
        "age": 26,
        "ability": 74,
        "form": 73
      },
      {
        "name": "Adrien Truffert",
        "position": "DEF",
        "age": 24,
        "ability": 78,
        "form": 79
      },
      {
        "name": "James Hill",
        "position": "DEF",
        "age": 24,
        "ability": 77,
        "form": 73
      },
      {
        "name": "Adam Smith",
        "position": "DEF",
        "age": 35,
        "ability": 72,
        "form": 63
      },
      {
        "name": "Bafodé Diakité",
        "position": "DEF",
        "age": 25,
        "ability": 73,
        "form": 64
      },
      {
        "name": "Marcus Tavernier",
        "position": "MID",
        "age": 27,
        "ability": 79,
        "form": 76
      },
      {
        "name": "Alex Scott",
        "position": "MID",
        "age": 22,
        "ability": 78,
        "form": 74
      },
      {
        "name": "Junior Kroupi",
        "position": "MID",
        "age": 20,
        "ability": 81,
        "form": 69
      },
      {
        "name": "Tyler Adams",
        "position": "MID",
        "age": 27,
        "ability": 75,
        "form": 68
      },
      {
        "name": "Rayan Vitor Simplício Rocha",
        "position": "MID",
        "age": 19,
        "ability": 79,
        "form": 75
      },
      {
        "name": "Francisco Evanilson de Lima Barbosa",
        "position": "FWD",
        "age": 26,
        "ability": 77,
        "form": 71
      },
      {
        "name": "Enes Ünal",
        "position": "FWD",
        "age": 29,
        "ability": 72,
        "form": 53
      },
      {
        "name": "Amine Adli",
        "position": "MID",
        "age": 26,
        "ability": 73,
        "form": 61
      },
      {
        "name": "David Brooks",
        "position": "MID",
        "age": 29,
        "ability": 73,
        "form": 59
      },
      {
        "name": "Ryan Christie",
        "position": "MID",
        "age": 31,
        "ability": 72,
        "form": 58
      },
      {
        "name": "Justin Kluivert",
        "position": "MID",
        "age": 27,
        "ability": 75,
        "form": 59
      },
      {
        "name": "Lewis Cook",
        "position": "MID",
        "age": 29,
        "ability": 73,
        "form": 60
      },
      {
        "name": "Veljko Milosavljevic",
        "position": "DEF",
        "age": 19,
        "ability": 73,
        "form": 62
      }
    ]
  },
  {
    "id": "brentford",
    "name": "Brentford",
    "shortName": "BRE",
    "city": "Brentford",
    "badgeUrl": "https://media.api-sports.io/football/teams/55.png",
    "colors": [
      "#e30613",
      "#fdb913"
    ],
    "strength": 75,
    "style": "Relentless",
    "players": [
      {
        "name": "Caoimhín Kelleher",
        "position": "GK",
        "age": 27,
        "ability": 76,
        "form": 76
      },
      {
        "name": "Nathan Collins",
        "position": "DEF",
        "age": 25,
        "ability": 77,
        "form": 74
      },
      {
        "name": "Michael Kayode",
        "position": "DEF",
        "age": 22,
        "ability": 74,
        "form": 71
      },
      {
        "name": "Sepp van den Berg",
        "position": "DEF",
        "age": 24,
        "ability": 75,
        "form": 73
      },
      {
        "name": "Kristoffer Ajer",
        "position": "DEF",
        "age": 28,
        "ability": 73,
        "form": 66
      },
      {
        "name": "Dango Ouattara",
        "position": "MID",
        "age": 24,
        "ability": 80,
        "form": 76
      },
      {
        "name": "Kevin Schade",
        "position": "MID",
        "age": 24,
        "ability": 78,
        "form": 73
      },
      {
        "name": "Keane Lewis-Potter",
        "position": "MID",
        "age": 25,
        "ability": 76,
        "form": 69
      },
      {
        "name": "Mikkel Damsgaard",
        "position": "MID",
        "age": 26,
        "ability": 76,
        "form": 70
      },
      {
        "name": "Yehor Yarmoliuk",
        "position": "MID",
        "age": 22,
        "ability": 75,
        "form": 68
      },
      {
        "name": "Igor Thiago Nascimento Rodrigues",
        "position": "FWD",
        "age": 25,
        "ability": 84,
        "form": 82
      },
      {
        "name": "Kaye Furo",
        "position": "FWD",
        "age": 19,
        "ability": 69,
        "form": 51
      },
      {
        "name": "Mathias Jensen",
        "position": "MID",
        "age": 30,
        "ability": 75,
        "form": 67
      },
      {
        "name": "Jordan Henderson",
        "position": "MID",
        "age": 36,
        "ability": 73,
        "form": 64
      },
      {
        "name": "Vitaly Janelt",
        "position": "MID",
        "age": 28,
        "ability": 74,
        "form": 66
      },
      {
        "name": "Rico Henry",
        "position": "DEF",
        "age": 29,
        "ability": 72,
        "form": 61
      },
      {
        "name": "Aaron Hickey",
        "position": "DEF",
        "age": 24,
        "ability": 71,
        "form": 57
      },
      {
        "name": "Ethan Pinnock",
        "position": "DEF",
        "age": 33,
        "ability": 73,
        "form": 67
      }
    ]
  },
  {
    "id": "brighton",
    "name": "Brighton & Hove Albion",
    "shortName": "BHA",
    "city": "Brighton",
    "badgeUrl": "https://media.api-sports.io/football/teams/51.png",
    "colors": [
      "#0057b8",
      "#ffffff"
    ],
    "strength": 76,
    "style": "Measured",
    "players": [
      {
        "name": "Bart Verbruggen",
        "position": "GK",
        "age": 23,
        "ability": 75,
        "form": 74
      },
      {
        "name": "Ferdi Kadıoğlu",
        "position": "DEF",
        "age": 26,
        "ability": 74,
        "form": 72
      },
      {
        "name": "Pascal Struijk",
        "position": "DEF",
        "age": 26,
        "ability": 75,
        "form": 71
      },
      {
        "name": "Lewis Dunk",
        "position": "DEF",
        "age": 34,
        "ability": 73,
        "form": 70
      },
      {
        "name": "Mats Wieffer",
        "position": "DEF",
        "age": 26,
        "ability": 75,
        "form": 71
      },
      {
        "name": "Yankuba Minteh",
        "position": "MID",
        "age": 22,
        "ability": 77,
        "form": 71
      },
      {
        "name": "Yasin Ayari",
        "position": "MID",
        "age": 22,
        "ability": 76,
        "form": 70
      },
      {
        "name": "Diego Gómez Amarilla",
        "position": "MID",
        "age": 23,
        "ability": 75,
        "form": 68
      },
      {
        "name": "Jack Hinshelwood",
        "position": "MID",
        "age": 21,
        "ability": 77,
        "form": 70
      },
      {
        "name": "Pascal Groß",
        "position": "MID",
        "age": 35,
        "ability": 77,
        "form": 74
      },
      {
        "name": "Danny Welbeck",
        "position": "FWD",
        "age": 35,
        "ability": 77,
        "form": 71
      },
      {
        "name": "Georginio Rutter",
        "position": "FWD",
        "age": 24,
        "ability": 75,
        "form": 66
      },
      {
        "name": "Maxim De Cuyper",
        "position": "DEF",
        "age": 25,
        "ability": 73,
        "form": 65
      },
      {
        "name": "Mitoma Kaoru",
        "position": "MID",
        "age": 29,
        "ability": 76,
        "form": 64
      },
      {
        "name": "Carlos Baleba",
        "position": "MID",
        "age": 22,
        "ability": 73,
        "form": 60
      },
      {
        "name": "Olivier Boscagli",
        "position": "DEF",
        "age": 28,
        "ability": 73,
        "form": 66
      },
      {
        "name": "Charalampos Kostoulas",
        "position": "FWD",
        "age": 19,
        "ability": 73,
        "form": 56
      },
      {
        "name": "Matt O'Riley",
        "position": "MID",
        "age": 25,
        "ability": 75,
        "form": 62
      }
    ]
  },
  {
    "id": "chelsea",
    "name": "Chelsea",
    "shortName": "CHE",
    "city": "Chelsea",
    "badgeUrl": "https://media.api-sports.io/football/teams/49.png",
    "colors": [
      "#034694",
      "#dba111"
    ],
    "strength": 83,
    "style": "Assertive",
    "players": [
      {
        "name": "Robert Lynch Sánchez",
        "position": "GK",
        "age": 28,
        "ability": 75,
        "form": 72
      },
      {
        "name": "Maxence Lacroix",
        "position": "DEF",
        "age": 26,
        "ability": 79,
        "form": 79
      },
      {
        "name": "Trevoh Chalobah",
        "position": "DEF",
        "age": 27,
        "ability": 77,
        "form": 76
      },
      {
        "name": "Reece James",
        "position": "DEF",
        "age": 26,
        "ability": 77,
        "form": 74
      },
      {
        "name": "Malo Gusto",
        "position": "DEF",
        "age": 23,
        "ability": 74,
        "form": 67
      },
      {
        "name": "Morgan Rogers",
        "position": "MID",
        "age": 24,
        "ability": 83,
        "form": 81
      },
      {
        "name": "Enzo Fernández",
        "position": "MID",
        "age": 25,
        "ability": 81,
        "form": 79
      },
      {
        "name": "Pedro Lomba Neto",
        "position": "MID",
        "age": 26,
        "ability": 79,
        "form": 74
      },
      {
        "name": "Cole Palmer",
        "position": "MID",
        "age": 24,
        "ability": 87,
        "form": 76
      },
      {
        "name": "Moisés Caicedo Corozo",
        "position": "MID",
        "age": 24,
        "ability": 76,
        "form": 72
      },
      {
        "name": "João Pedro Junqueira de Jesus",
        "position": "FWD",
        "age": 24,
        "ability": 83,
        "form": 82
      },
      {
        "name": "Liam Delap",
        "position": "FWD",
        "age": 23,
        "ability": 73,
        "form": 57
      },
      {
        "name": "Wesley Fofana",
        "position": "DEF",
        "age": 25,
        "ability": 74,
        "form": 65
      },
      {
        "name": "Estêvão Almeida de Oliveira Gonçalves",
        "position": "MID",
        "age": 19,
        "ability": 77,
        "form": 63
      },
      {
        "name": "Josh Acheampong",
        "position": "DEF",
        "age": 20,
        "ability": 72,
        "form": 60
      },
      {
        "name": "Tosin Adarabioyo",
        "position": "DEF",
        "age": 28,
        "ability": 71,
        "form": 60
      },
      {
        "name": "Jamie Bynoe-Gittens",
        "position": "MID",
        "age": 21,
        "ability": 75,
        "form": 56
      },
      {
        "name": "Jorrel Hato",
        "position": "DEF",
        "age": 20,
        "ability": 70,
        "form": 53
      }
    ]
  },
  {
    "id": "coventry",
    "name": "Coventry City",
    "shortName": "COV",
    "city": "Coventry",
    "badgeUrl": "https://media.api-sports.io/football/teams/134.png",
    "colors": [
      "#1d428a",
      "#ffffff"
    ],
    "strength": 70,
    "style": "Measured",
    "players": [
      {
        "name": "Ben Wilson",
        "position": "GK",
        "age": 33,
        "ability": 68,
        "form": 45
      },
      {
        "name": "Bobby Thomas",
        "position": "DEF",
        "age": 25,
        "ability": 67,
        "form": 45
      },
      {
        "name": "Liam Kitching",
        "position": "DEF",
        "age": 26,
        "ability": 67,
        "form": 45
      },
      {
        "name": "Milan van Ewijk",
        "position": "DEF",
        "age": 25,
        "ability": 67,
        "form": 45
      },
      {
        "name": "Jay Dasilva",
        "position": "DEF",
        "age": 28,
        "ability": 67,
        "form": 45
      },
      {
        "name": "Frank Onyeka",
        "position": "MID",
        "age": 28,
        "ability": 72,
        "form": 57
      },
      {
        "name": "Ephron Mason-Clark",
        "position": "MID",
        "age": 26,
        "ability": 71,
        "form": 45
      },
      {
        "name": "Victor Torp",
        "position": "MID",
        "age": 27,
        "ability": 71,
        "form": 45
      },
      {
        "name": "Loum Tchaouna",
        "position": "MID",
        "age": 22,
        "ability": 71,
        "form": 45
      },
      {
        "name": "Jack Rudoni",
        "position": "MID",
        "age": 25,
        "ability": 70,
        "form": 45
      },
      {
        "name": "Haji Wright",
        "position": "FWD",
        "age": 28,
        "ability": 71,
        "form": 45
      },
      {
        "name": "Brandon Thomas-Asante",
        "position": "FWD",
        "age": 27,
        "ability": 70,
        "form": 45
      },
      {
        "name": "Matt Grimes",
        "position": "MID",
        "age": 31,
        "ability": 70,
        "form": 45
      },
      {
        "name": "Tatsuhiro Sakamoto",
        "position": "MID",
        "age": 29,
        "ability": 70,
        "form": 45
      },
      {
        "name": "Josh Eccles",
        "position": "MID",
        "age": 26,
        "ability": 70,
        "form": 45
      },
      {
        "name": "Ellis Simms",
        "position": "FWD",
        "age": 25,
        "ability": 70,
        "form": 45
      },
      {
        "name": "Jahnoah Markelo",
        "position": "FWD",
        "age": 23,
        "ability": 70,
        "form": 45
      },
      {
        "name": "George Shepherd",
        "position": "MID",
        "age": 20,
        "ability": 68,
        "form": 45
      }
    ]
  },
  {
    "id": "crystal-palace",
    "name": "Crystal Palace",
    "shortName": "CRY",
    "city": "Crystal Palace",
    "badgeUrl": "https://media.api-sports.io/football/teams/52.png",
    "colors": [
      "#1b458f",
      "#c4122e"
    ],
    "strength": 75,
    "style": "Relentless",
    "players": [
      {
        "name": "Dean Henderson",
        "position": "GK",
        "age": 29,
        "ability": 76,
        "form": 74
      },
      {
        "name": "Daniel Muñoz Mejía",
        "position": "DEF",
        "age": 30,
        "ability": 78,
        "form": 79
      },
      {
        "name": "Tyrick Mitchell",
        "position": "DEF",
        "age": 26,
        "ability": 75,
        "form": 74
      },
      {
        "name": "Chris Richards",
        "position": "DEF",
        "age": 26,
        "ability": 76,
        "form": 75
      },
      {
        "name": "Jaydee Canvot",
        "position": "DEF",
        "age": 20,
        "ability": 75,
        "form": 70
      },
      {
        "name": "Ismaïla Sarr",
        "position": "MID",
        "age": 28,
        "ability": 80,
        "form": 75
      },
      {
        "name": "Adam Wharton",
        "position": "MID",
        "age": 22,
        "ability": 76,
        "form": 71
      },
      {
        "name": "Yéremy Pino Santos",
        "position": "MID",
        "age": 23,
        "ability": 75,
        "form": 64
      },
      {
        "name": "Jefferson Lerma Solís",
        "position": "MID",
        "age": 31,
        "ability": 73,
        "form": 62
      },
      {
        "name": "Daichi Kamada",
        "position": "MID",
        "age": 29,
        "ability": 73,
        "form": 64
      },
      {
        "name": "Jean-Philippe Mateta",
        "position": "FWD",
        "age": 29,
        "ability": 79,
        "form": 72
      },
      {
        "name": "Jørgen Strand Larsen",
        "position": "FWD",
        "age": 26,
        "ability": 76,
        "form": 63
      },
      {
        "name": "Brennan Johnson",
        "position": "MID",
        "age": 25,
        "ability": 75,
        "form": 61
      },
      {
        "name": "Will Hughes",
        "position": "MID",
        "age": 31,
        "ability": 71,
        "form": 58
      },
      {
        "name": "Justin Devenny",
        "position": "MID",
        "age": 22,
        "ability": 72,
        "form": 57
      },
      {
        "name": "Eddie Nketiah",
        "position": "FWD",
        "age": 27,
        "ability": 73,
        "form": 57
      },
      {
        "name": "Chadi Riad Dnanou",
        "position": "DEF",
        "age": 23,
        "ability": 71,
        "form": 60
      },
      {
        "name": "Christantus Uche",
        "position": "FWD",
        "age": 22,
        "ability": 71,
        "form": 53
      }
    ]
  },
  {
    "id": "everton",
    "name": "Everton",
    "shortName": "EVE",
    "city": "Everton",
    "badgeUrl": "https://media.api-sports.io/football/teams/45.png",
    "colors": [
      "#003399",
      "#ffffff"
    ],
    "strength": 73,
    "style": "Assertive",
    "players": [
      {
        "name": "Jordan Pickford",
        "position": "GK",
        "age": 32,
        "ability": 77,
        "form": 75
      },
      {
        "name": "James Tarkowski",
        "position": "DEF",
        "age": 33,
        "ability": 79,
        "form": 81
      },
      {
        "name": "Michael Keane",
        "position": "DEF",
        "age": 33,
        "ability": 76,
        "form": 75
      },
      {
        "name": "Jake O'Brien",
        "position": "DEF",
        "age": 25,
        "ability": 75,
        "form": 71
      },
      {
        "name": "Vitalii Mykolenko",
        "position": "DEF",
        "age": 27,
        "ability": 73,
        "form": 70
      },
      {
        "name": "James Garner",
        "position": "MID",
        "age": 25,
        "ability": 79,
        "form": 78
      },
      {
        "name": "Kiernan Dewsbury-Hall",
        "position": "MID",
        "age": 27,
        "ability": 81,
        "form": 81
      },
      {
        "name": "Iliman Ndiaye",
        "position": "MID",
        "age": 26,
        "ability": 79,
        "form": 76
      },
      {
        "name": "Tim Iroegbunam",
        "position": "MID",
        "age": 23,
        "ability": 73,
        "form": 62
      },
      {
        "name": "Dwight McNeil",
        "position": "MID",
        "age": 26,
        "ability": 75,
        "form": 62
      },
      {
        "name": "Norberto Bercique Gomes Betuncal",
        "position": "FWD",
        "age": 28,
        "ability": 75,
        "form": 66
      },
      {
        "name": "Thierno Barry",
        "position": "FWD",
        "age": 23,
        "ability": 75,
        "form": 65
      },
      {
        "name": "Merlin Röhl",
        "position": "MID",
        "age": 24,
        "ability": 73,
        "form": 60
      },
      {
        "name": "Jarrad Branthwaite",
        "position": "DEF",
        "age": 24,
        "ability": 75,
        "form": 66
      },
      {
        "name": "Carlos Alcaraz Durán",
        "position": "MID",
        "age": 23,
        "ability": 72,
        "form": 55
      },
      {
        "name": "Nathan Patterson",
        "position": "DEF",
        "age": 24,
        "ability": 72,
        "form": 64
      },
      {
        "name": "Harrison Armstrong",
        "position": "MID",
        "age": 19,
        "ability": 72,
        "form": 56
      },
      {
        "name": "Tyler Dibling",
        "position": "MID",
        "age": 20,
        "ability": 73,
        "form": 54
      }
    ]
  },
  {
    "id": "fulham",
    "name": "Fulham",
    "shortName": "FUL",
    "city": "Fulham",
    "badgeUrl": "https://media.api-sports.io/football/teams/36.png",
    "colors": [
      "#000000",
      "#ffffff"
    ],
    "strength": 72,
    "style": "Measured",
    "players": [
      {
        "name": "Bernd Leno",
        "position": "GK",
        "age": 34,
        "ability": 74,
        "form": 72
      },
      {
        "name": "Joachim Andersen",
        "position": "DEF",
        "age": 30,
        "ability": 76,
        "form": 74
      },
      {
        "name": "Calvin Bassey",
        "position": "DEF",
        "age": 26,
        "ability": 74,
        "form": 72
      },
      {
        "name": "Ryan Sessegnon",
        "position": "DEF",
        "age": 26,
        "ability": 73,
        "form": 70
      },
      {
        "name": "Kenny Tete",
        "position": "DEF",
        "age": 30,
        "ability": 74,
        "form": 72
      },
      {
        "name": "Alex Iwobi",
        "position": "MID",
        "age": 30,
        "ability": 76,
        "form": 72
      },
      {
        "name": "Sander Berge",
        "position": "MID",
        "age": 28,
        "ability": 75,
        "form": 68
      },
      {
        "name": "Emile Smith Rowe",
        "position": "MID",
        "age": 26,
        "ability": 75,
        "form": 63
      },
      {
        "name": "Saša Lukić",
        "position": "MID",
        "age": 29,
        "ability": 74,
        "form": 65
      },
      {
        "name": "Josh King",
        "position": "MID",
        "age": 19,
        "ability": 74,
        "form": 60
      },
      {
        "name": "Rodrigo Muniz Carvalho",
        "position": "FWD",
        "age": 25,
        "ability": 73,
        "form": 58
      },
      {
        "name": "Jonah Kusi-Asare",
        "position": "FWD",
        "age": 19,
        "ability": 69,
        "form": 51
      },
      {
        "name": "Antonee Robinson",
        "position": "DEF",
        "age": 28,
        "ability": 73,
        "form": 67
      },
      {
        "name": "Timothy Castagne",
        "position": "DEF",
        "age": 30,
        "ability": 72,
        "form": 61
      },
      {
        "name": "Kevin Santos Lopes de Macedo",
        "position": "MID",
        "age": 23,
        "ability": 74,
        "form": 59
      },
      {
        "name": "Oscar Bobb",
        "position": "MID",
        "age": 23,
        "ability": 73,
        "form": 58
      },
      {
        "name": "Tom Cairney",
        "position": "MID",
        "age": 35,
        "ability": 72,
        "form": 57
      },
      {
        "name": "Jorge Cuenca Barreno",
        "position": "DEF",
        "age": 26,
        "ability": 72,
        "form": 63
      }
    ]
  },
  {
    "id": "hull-city",
    "name": "Hull City",
    "shortName": "HUL",
    "city": "Hull",
    "badgeUrl": "https://media.api-sports.io/football/teams/322.png",
    "colors": [
      "#f5a800",
      "#000000"
    ],
    "strength": 69,
    "style": "Assertive",
    "players": [
      {
        "name": "Jack Butland",
        "position": "GK",
        "age": 33,
        "ability": 68,
        "form": 45
      },
      {
        "name": "John Egan",
        "position": "DEF",
        "age": 33,
        "ability": 67,
        "form": 45
      },
      {
        "name": "Charlie Hughes",
        "position": "DEF",
        "age": 22,
        "ability": 67,
        "form": 45
      },
      {
        "name": "Semi Ajayi",
        "position": "DEF",
        "age": 32,
        "ability": 67,
        "form": 45
      },
      {
        "name": "Lewie Coyle",
        "position": "DEF",
        "age": 30,
        "ability": 67,
        "form": 45
      },
      {
        "name": "Mohamed Belloumi",
        "position": "MID",
        "age": 24,
        "ability": 70,
        "form": 45
      },
      {
        "name": "Liam Millar",
        "position": "MID",
        "age": 26,
        "ability": 70,
        "form": 45
      },
      {
        "name": "Abdülkadir Ömür",
        "position": "MID",
        "age": 27,
        "ability": 70,
        "form": 45
      },
      {
        "name": "Abu Kamara",
        "position": "MID",
        "age": 23,
        "ability": 70,
        "form": 45
      },
      {
        "name": "David Akintola",
        "position": "MID",
        "age": 30,
        "ability": 70,
        "form": 45
      },
      {
        "name": "Oli McBurnie",
        "position": "FWD",
        "age": 30,
        "ability": 71,
        "form": 45
      },
      {
        "name": "Enis Destan",
        "position": "FWD",
        "age": 24,
        "ability": 68,
        "form": 45
      },
      {
        "name": "Hidemasa Morita",
        "position": "MID",
        "age": 31,
        "ability": 70,
        "form": 45
      },
      {
        "name": "Kieran Dowell",
        "position": "MID",
        "age": 28,
        "ability": 68,
        "form": 45
      },
      {
        "name": "Matt Crooks",
        "position": "MID",
        "age": 32,
        "ability": 68,
        "form": 45
      },
      {
        "name": "Regan Slater",
        "position": "MID",
        "age": 26,
        "ability": 68,
        "form": 45
      },
      {
        "name": "Eliot Matazo",
        "position": "MID",
        "age": 24,
        "ability": 68,
        "form": 45
      },
      {
        "name": "Óscar Zambrano",
        "position": "MID",
        "age": 22,
        "ability": 68,
        "form": 45
      }
    ]
  },
  {
    "id": "ipswich-town",
    "name": "Ipswich Town",
    "shortName": "IPS",
    "city": "Ipswich",
    "badgeUrl": "https://media.api-sports.io/football/teams/57.png",
    "colors": [
      "#3b5da8",
      "#ffffff"
    ],
    "strength": 68,
    "style": "Relentless",
    "players": [
      {
        "name": "Christian Walton",
        "position": "GK",
        "age": 30,
        "ability": 68,
        "form": 45
      },
      {
        "name": "Issa Diop",
        "position": "DEF",
        "age": 29,
        "ability": 71,
        "form": 61
      },
      {
        "name": "Cédric Kipré",
        "position": "DEF",
        "age": 29,
        "ability": 67,
        "form": 45
      },
      {
        "name": "Dara O'Shea",
        "position": "DEF",
        "age": 27,
        "ability": 67,
        "form": 45
      },
      {
        "name": "Leif Davis",
        "position": "DEF",
        "age": 26,
        "ability": 67,
        "form": 45
      },
      {
        "name": "Jack Clarke",
        "position": "MID",
        "age": 25,
        "ability": 71,
        "form": 45
      },
      {
        "name": "Abdul Fatawu",
        "position": "MID",
        "age": 22,
        "ability": 71,
        "form": 45
      },
      {
        "name": "Jaden Philogene",
        "position": "MID",
        "age": 24,
        "ability": 71,
        "form": 45
      },
      {
        "name": "Daizen Maeda",
        "position": "MID",
        "age": 28,
        "ability": 71,
        "form": 45
      },
      {
        "name": "Marcelino Núñez",
        "position": "MID",
        "age": 26,
        "ability": 70,
        "form": 45
      },
      {
        "name": "Emersonn Correia da Silva",
        "position": "FWD",
        "age": 22,
        "ability": 71,
        "form": 45
      },
      {
        "name": "George Hirst",
        "position": "FWD",
        "age": 27,
        "ability": 70,
        "form": 45
      },
      {
        "name": "Azor Matusiwa",
        "position": "MID",
        "age": 28,
        "ability": 70,
        "form": 45
      },
      {
        "name": "Wes Burns",
        "position": "MID",
        "age": 31,
        "ability": 70,
        "form": 45
      },
      {
        "name": "Chiedozie Ogbene",
        "position": "MID",
        "age": 29,
        "ability": 70,
        "form": 45
      },
      {
        "name": "Sam Szmodics",
        "position": "MID",
        "age": 30,
        "ability": 70,
        "form": 45
      },
      {
        "name": "Chuba Akpom",
        "position": "FWD",
        "age": 30,
        "ability": 70,
        "form": 45
      },
      {
        "name": "Ali Al-Hamadi",
        "position": "FWD",
        "age": 24,
        "ability": 70,
        "form": 45
      }
    ]
  },
  {
    "id": "leeds",
    "name": "Leeds United",
    "shortName": "LEE",
    "city": "Leeds",
    "badgeUrl": "https://media.api-sports.io/football/teams/63.png",
    "colors": [
      "#ffcd00",
      "#1d428a"
    ],
    "strength": 75,
    "style": "Relentless",
    "players": [
      {
        "name": "Lucas Estella Perri",
        "position": "GK",
        "age": 28,
        "ability": 72,
        "form": 65
      },
      {
        "name": "Joe Rodon",
        "position": "DEF",
        "age": 28,
        "ability": 74,
        "form": 71
      },
      {
        "name": "Jaka Bijol",
        "position": "DEF",
        "age": 27,
        "ability": 76,
        "form": 74
      },
      {
        "name": "Jayden Bogle",
        "position": "DEF",
        "age": 26,
        "ability": 73,
        "form": 69
      },
      {
        "name": "James Justin",
        "position": "DEF",
        "age": 28,
        "ability": 73,
        "form": 69
      },
      {
        "name": "Harry Wilson",
        "position": "MID",
        "age": 29,
        "ability": 80,
        "form": 80
      },
      {
        "name": "Anton Stach",
        "position": "MID",
        "age": 27,
        "ability": 79,
        "form": 79
      },
      {
        "name": "Ethan Ampadu",
        "position": "MID",
        "age": 25,
        "ability": 77,
        "form": 75
      },
      {
        "name": "Brenden Aaronson",
        "position": "MID",
        "age": 25,
        "ability": 76,
        "form": 71
      },
      {
        "name": "Noah Okafor",
        "position": "MID",
        "age": 26,
        "ability": 77,
        "form": 72
      },
      {
        "name": "Dominic Calvert-Lewin",
        "position": "FWD",
        "age": 29,
        "ability": 79,
        "form": 76
      },
      {
        "name": "Lukas Nmecha",
        "position": "FWD",
        "age": 27,
        "ability": 75,
        "form": 62
      },
      {
        "name": "Gabriel Gudmundsson",
        "position": "DEF",
        "age": 27,
        "ability": 72,
        "form": 64
      },
      {
        "name": "Tanaka Ao",
        "position": "MID",
        "age": 27,
        "ability": 73,
        "form": 63
      },
      {
        "name": "Ilia Gruev",
        "position": "MID",
        "age": 26,
        "ability": 73,
        "form": 63
      },
      {
        "name": "Sean Longstaff",
        "position": "MID",
        "age": 28,
        "ability": 73,
        "form": 62
      },
      {
        "name": "Wilfried Gnonto",
        "position": "MID",
        "age": 22,
        "ability": 72,
        "form": 55
      },
      {
        "name": "Sebastiaan Bornauw",
        "position": "DEF",
        "age": 27,
        "ability": 71,
        "form": 59
      }
    ]
  },
  {
    "id": "liverpool",
    "name": "Liverpool",
    "shortName": "LIV",
    "city": "Liverpool",
    "badgeUrl": "https://media.api-sports.io/football/teams/40.png",
    "colors": [
      "#c8102e",
      "#00b2a9"
    ],
    "strength": 87,
    "style": "Relentless",
    "players": [
      {
        "name": "Alisson Becker",
        "position": "GK",
        "age": 33,
        "ability": 76,
        "form": 72
      },
      {
        "name": "Virgil van Dijk",
        "position": "DEF",
        "age": 35,
        "ability": 81,
        "form": 81
      },
      {
        "name": "Milos Kerkez",
        "position": "DEF",
        "age": 22,
        "ability": 75,
        "form": 65
      },
      {
        "name": "Jeremie Frimpong",
        "position": "DEF",
        "age": 25,
        "ability": 75,
        "form": 66
      },
      {
        "name": "Conor Bradley",
        "position": "DEF",
        "age": 23,
        "ability": 73,
        "form": 64
      },
      {
        "name": "Dominik Szoboszlai",
        "position": "MID",
        "age": 25,
        "ability": 81,
        "form": 79
      },
      {
        "name": "Ryan Gravenberch",
        "position": "MID",
        "age": 24,
        "ability": 79,
        "form": 76
      },
      {
        "name": "Cody Gakpo",
        "position": "MID",
        "age": 27,
        "ability": 80,
        "form": 73
      },
      {
        "name": "Florian Wirtz",
        "position": "MID",
        "age": 23,
        "ability": 81,
        "form": 74
      },
      {
        "name": "Alexis Mac Allister",
        "position": "MID",
        "age": 27,
        "ability": 76,
        "form": 69
      },
      {
        "name": "Hugo Ekitiké",
        "position": "FWD",
        "age": 24,
        "ability": 82,
        "form": 76
      },
      {
        "name": "Alexander Isak",
        "position": "FWD",
        "age": 26,
        "ability": 83,
        "form": 64
      },
      {
        "name": "Curtis Jones",
        "position": "MID",
        "age": 25,
        "ability": 75,
        "form": 64
      },
      {
        "name": "Rio Ngumoha",
        "position": "MID",
        "age": 17,
        "ability": 75,
        "form": 60
      },
      {
        "name": "Federico Chiesa",
        "position": "MID",
        "age": 28,
        "ability": 73,
        "form": 54
      },
      {
        "name": "Giorgi Mamardashvili",
        "position": "GK",
        "age": 25,
        "ability": 74,
        "form": 68
      },
      {
        "name": "Joe Gomez",
        "position": "DEF",
        "age": 29,
        "ability": 72,
        "form": 56
      },
      {
        "name": "Endo Wataru",
        "position": "MID",
        "age": 33,
        "ability": 72,
        "form": 54
      }
    ]
  },
  {
    "id": "manchester-city",
    "name": "Manchester City",
    "shortName": "MCI",
    "city": "Manchester",
    "badgeUrl": "https://media.api-sports.io/football/teams/50.png",
    "colors": [
      "#6cabdd",
      "#ffffff"
    ],
    "strength": 90,
    "style": "Measured",
    "players": [
      {
        "name": "Gianluigi Donnarumma",
        "position": "GK",
        "age": 27,
        "ability": 77,
        "form": 76
      },
      {
        "name": "Marc Guéhi",
        "position": "DEF",
        "age": 26,
        "ability": 80,
        "form": 83
      },
      {
        "name": "Nico O'Reilly",
        "position": "DEF",
        "age": 21,
        "ability": 80,
        "form": 80
      },
      {
        "name": "Matheus Nunes",
        "position": "DEF",
        "age": 27,
        "ability": 79,
        "form": 79
      },
      {
        "name": "Rúben dos Santos Gato Alves Dias",
        "position": "DEF",
        "age": 29,
        "ability": 77,
        "form": 76
      },
      {
        "name": "Antoine Semenyo",
        "position": "MID",
        "age": 26,
        "ability": 87,
        "form": 86
      },
      {
        "name": "Elliot Anderson",
        "position": "MID",
        "age": 23,
        "ability": 81,
        "form": 81
      },
      {
        "name": "Rayan Cherki",
        "position": "MID",
        "age": 22,
        "ability": 81,
        "form": 74
      },
      {
        "name": "Phil Foden",
        "position": "MID",
        "age": 26,
        "ability": 80,
        "form": 74
      },
      {
        "name": "Jérémy Doku",
        "position": "MID",
        "age": 24,
        "ability": 81,
        "form": 73
      },
      {
        "name": "Erling Haaland",
        "position": "FWD",
        "age": 26,
        "ability": 87,
        "form": 92
      },
      {
        "name": "Omar Marmoush",
        "position": "FWD",
        "age": 27,
        "ability": 78,
        "form": 63
      },
      {
        "name": "Tijjani Reijnders",
        "position": "MID",
        "age": 28,
        "ability": 77,
        "form": 69
      },
      {
        "name": "Jack Grealish",
        "position": "MID",
        "age": 30,
        "ability": 79,
        "form": 73
      },
      {
        "name": "Joško Gvardiol",
        "position": "DEF",
        "age": 24,
        "ability": 77,
        "form": 75
      },
      {
        "name": "Rodrigo 'Rodri' Hernandez Cascante",
        "position": "MID",
        "age": 30,
        "ability": 78,
        "form": 68
      },
      {
        "name": "Abdukodir Khusanov",
        "position": "DEF",
        "age": 22,
        "ability": 76,
        "form": 68
      },
      {
        "name": "Nico González Iglesias",
        "position": "MID",
        "age": 24,
        "ability": 75,
        "form": 64
      }
    ]
  },
  {
    "id": "manchester-united",
    "name": "Manchester United",
    "shortName": "MUN",
    "city": "Manchester",
    "badgeUrl": "https://media.api-sports.io/football/teams/33.png",
    "colors": [
      "#da291c",
      "#fbe122"
    ],
    "strength": 82,
    "style": "Assertive",
    "players": [
      {
        "name": "Senne Lammens",
        "position": "GK",
        "age": 24,
        "ability": 75,
        "form": 72
      },
      {
        "name": "Luke Shaw",
        "position": "DEF",
        "age": 31,
        "ability": 74,
        "form": 71
      },
      {
        "name": "Diogo Dalot Teixeira",
        "position": "DEF",
        "age": 27,
        "ability": 75,
        "form": 71
      },
      {
        "name": "Harry Maguire",
        "position": "DEF",
        "age": 33,
        "ability": 75,
        "form": 72
      },
      {
        "name": "Leny Yoro",
        "position": "DEF",
        "age": 20,
        "ability": 73,
        "form": 60
      },
      {
        "name": "Bruno Borges Fernandes",
        "position": "MID",
        "age": 31,
        "ability": 87,
        "form": 92
      },
      {
        "name": "Bryan Mbeumo",
        "position": "MID",
        "age": 26,
        "ability": 84,
        "form": 78
      },
      {
        "name": "Matheus Santos Carneiro da Cunha",
        "position": "MID",
        "age": 27,
        "ability": 84,
        "form": 77
      },
      {
        "name": "Patrick Dorgu",
        "position": "MID",
        "age": 21,
        "ability": 77,
        "form": 71
      },
      {
        "name": "Amad Diallo",
        "position": "MID",
        "age": 24,
        "ability": 77,
        "form": 67
      },
      {
        "name": "Benjamin Sesko",
        "position": "FWD",
        "age": 23,
        "ability": 80,
        "form": 71
      },
      {
        "name": "Joshua Zirkzee",
        "position": "FWD",
        "age": 25,
        "ability": 73,
        "form": 57
      },
      {
        "name": "Kobbie Mainoo",
        "position": "MID",
        "age": 21,
        "ability": 75,
        "form": 65
      },
      {
        "name": "Karl Darlow",
        "position": "GK",
        "age": 35,
        "ability": 73,
        "form": 69
      },
      {
        "name": "Youri Tielemans",
        "position": "MID",
        "age": 29,
        "ability": 76,
        "form": 66
      },
      {
        "name": "Mason Mount",
        "position": "MID",
        "age": 27,
        "ability": 75,
        "form": 62
      },
      {
        "name": "Noussair Mazraoui",
        "position": "DEF",
        "age": 28,
        "ability": 72,
        "form": 63
      },
      {
        "name": "Lisandro Martínez",
        "position": "DEF",
        "age": 28,
        "ability": 73,
        "form": 65
      }
    ]
  },
  {
    "id": "newcastle",
    "name": "Newcastle United",
    "shortName": "NEW",
    "city": "Newcastle",
    "badgeUrl": "https://media.api-sports.io/football/teams/34.png",
    "colors": [
      "#241f20",
      "#ffffff"
    ],
    "strength": 80,
    "style": "Relentless",
    "players": [
      {
        "name": "Nick Pope",
        "position": "GK",
        "age": 34,
        "ability": 75,
        "form": 72
      },
      {
        "name": "Malick Thiaw",
        "position": "DEF",
        "age": 24,
        "ability": 76,
        "form": 74
      },
      {
        "name": "Dan Burn",
        "position": "DEF",
        "age": 34,
        "ability": 75,
        "form": 70
      },
      {
        "name": "Sven Botman",
        "position": "DEF",
        "age": 26,
        "ability": 75,
        "form": 71
      },
      {
        "name": "Lewis Hall",
        "position": "DEF",
        "age": 21,
        "ability": 74,
        "form": 66
      },
      {
        "name": "Bruno Guimarães Rodriguez Moura",
        "position": "MID",
        "age": 28,
        "ability": 82,
        "form": 83
      },
      {
        "name": "Harvey Barnes",
        "position": "MID",
        "age": 28,
        "ability": 77,
        "form": 67
      },
      {
        "name": "Jacob Murphy",
        "position": "MID",
        "age": 31,
        "ability": 76,
        "form": 64
      },
      {
        "name": "Joelinton Cássio Apolinário de Lira",
        "position": "MID",
        "age": 29,
        "ability": 75,
        "form": 66
      },
      {
        "name": "Lewis Miley",
        "position": "MID",
        "age": 20,
        "ability": 76,
        "form": 67
      },
      {
        "name": "Nick Woltemade",
        "position": "FWD",
        "age": 24,
        "ability": 77,
        "form": 69
      },
      {
        "name": "William Osula",
        "position": "FWD",
        "age": 22,
        "ability": 76,
        "form": 66
      },
      {
        "name": "Jacob Ramsey",
        "position": "MID",
        "age": 25,
        "ability": 73,
        "form": 62
      },
      {
        "name": "Tino Livramento",
        "position": "DEF",
        "age": 23,
        "ability": 75,
        "form": 69
      },
      {
        "name": "Fabian Schär",
        "position": "DEF",
        "age": 34,
        "ability": 74,
        "form": 67
      },
      {
        "name": "Anthony Elanga",
        "position": "MID",
        "age": 24,
        "ability": 75,
        "form": 57
      },
      {
        "name": "Joe Willock",
        "position": "MID",
        "age": 26,
        "ability": 72,
        "form": 57
      },
      {
        "name": "Yoane Wissa",
        "position": "FWD",
        "age": 29,
        "ability": 74,
        "form": 55
      }
    ]
  },
  {
    "id": "nottingham-forest",
    "name": "Nottingham Forest",
    "shortName": "NFO",
    "city": "Nottingham Forest",
    "badgeUrl": "https://media.api-sports.io/football/teams/65.png",
    "colors": [
      "#e53233",
      "#ffffff"
    ],
    "strength": 77,
    "style": "Measured",
    "players": [
      {
        "name": "Matz Sels",
        "position": "GK",
        "age": 34,
        "ability": 75,
        "form": 72
      },
      {
        "name": "Neco Williams",
        "position": "DEF",
        "age": 25,
        "ability": 76,
        "form": 74
      },
      {
        "name": "Nikola Milenković",
        "position": "DEF",
        "age": 28,
        "ability": 76,
        "form": 72
      },
      {
        "name": "Murillo Costa dos Santos",
        "position": "DEF",
        "age": 24,
        "ability": 76,
        "form": 70
      },
      {
        "name": "Ola Aina",
        "position": "DEF",
        "age": 29,
        "ability": 74,
        "form": 71
      },
      {
        "name": "Morgan Gibbs-White",
        "position": "MID",
        "age": 26,
        "ability": 85,
        "form": 83
      },
      {
        "name": "Ibrahim Sangaré",
        "position": "MID",
        "age": 28,
        "ability": 75,
        "form": 69
      },
      {
        "name": "Callum Hudson-Odoi",
        "position": "MID",
        "age": 25,
        "ability": 77,
        "form": 67
      },
      {
        "name": "Omari Hutchinson",
        "position": "MID",
        "age": 22,
        "ability": 75,
        "form": 65
      },
      {
        "name": "Dan Ndoye",
        "position": "MID",
        "age": 25,
        "ability": 74,
        "form": 60
      },
      {
        "name": "Igor Jesus Maciel da Cruz",
        "position": "FWD",
        "age": 25,
        "ability": 77,
        "form": 69
      },
      {
        "name": "Taiwo Awoniyi",
        "position": "FWD",
        "age": 28,
        "ability": 74,
        "form": 61
      },
      {
        "name": "Felipe Rodrigues da Silva",
        "position": "DEF",
        "age": 25,
        "ability": 73,
        "form": 63
      },
      {
        "name": "Nicolás Domínguez",
        "position": "MID",
        "age": 28,
        "ability": 72,
        "form": 59
      },
      {
        "name": "Chris Wood",
        "position": "FWD",
        "age": 34,
        "ability": 76,
        "form": 63
      },
      {
        "name": "Nicolò Savona",
        "position": "DEF",
        "age": 23,
        "ability": 72,
        "form": 64
      },
      {
        "name": "Ryan Yates",
        "position": "MID",
        "age": 28,
        "ability": 71,
        "form": 55
      },
      {
        "name": "Dilane Bakwa",
        "position": "MID",
        "age": 23,
        "ability": 73,
        "form": 58
      }
    ]
  },
  {
    "id": "tottenham-hotspur",
    "name": "Tottenham Hotspur",
    "shortName": "TOT",
    "city": "Tottenham Hotspur",
    "badgeUrl": "https://media.api-sports.io/football/teams/47.png",
    "colors": [
      "#132257",
      "#ffffff"
    ],
    "strength": 79,
    "style": "Assertive",
    "players": [
      {
        "name": "Martin Dubravka",
        "position": "GK",
        "age": 37,
        "ability": 72,
        "form": 69
      },
      {
        "name": "Marcos Senesi Barón",
        "position": "DEF",
        "age": 29,
        "ability": 80,
        "form": 81
      },
      {
        "name": "Jan Paul van Hecke",
        "position": "DEF",
        "age": 26,
        "ability": 76,
        "form": 77
      },
      {
        "name": "Pedro Porro Sauceda",
        "position": "DEF",
        "age": 26,
        "ability": 76,
        "form": 72
      },
      {
        "name": "Micky van de Ven",
        "position": "DEF",
        "age": 25,
        "ability": 75,
        "form": 72
      },
      {
        "name": "Mateus Fernandes",
        "position": "MID",
        "age": 22,
        "ability": 79,
        "form": 75
      },
      {
        "name": "Rodrigo Bentancur",
        "position": "MID",
        "age": 29,
        "ability": 76,
        "form": 69
      },
      {
        "name": "Sandro Tonali",
        "position": "MID",
        "age": 26,
        "ability": 75,
        "form": 65
      },
      {
        "name": "Xavi Simons",
        "position": "MID",
        "age": 23,
        "ability": 76,
        "form": 67
      },
      {
        "name": "Mathys Tel",
        "position": "MID",
        "age": 21,
        "ability": 76,
        "form": 63
      },
      {
        "name": "Richarlison de Andrade",
        "position": "FWD",
        "age": 29,
        "ability": 77,
        "form": 72
      },
      {
        "name": "Dominic Solanke-Mitchell",
        "position": "FWD",
        "age": 28,
        "ability": 76,
        "form": 64
      },
      {
        "name": "Cristian Romero",
        "position": "DEF",
        "age": 28,
        "ability": 76,
        "form": 74
      },
      {
        "name": "Guglielmo Vicario",
        "position": "GK",
        "age": 29,
        "ability": 73,
        "form": 69
      },
      {
        "name": "Djed Spence",
        "position": "DEF",
        "age": 25,
        "ability": 73,
        "form": 66
      },
      {
        "name": "Mohammed Kudus",
        "position": "MID",
        "age": 26,
        "ability": 79,
        "form": 72
      },
      {
        "name": "Pape Matar Sarr",
        "position": "MID",
        "age": 23,
        "ability": 73,
        "form": 65
      },
      {
        "name": "Kevin Danso",
        "position": "DEF",
        "age": 27,
        "ability": 73,
        "form": 64
      }
    ]
  },
  {
    "id": "sunderland",
    "name": "Sunderland",
    "shortName": "SUN",
    "city": "Sunderland",
    "badgeUrl": "https://media.api-sports.io/football/teams/746.png",
    "colors": [
      "#eb172b",
      "#ffffff"
    ],
    "strength": 69,
    "style": "Assertive",
    "players": [
      {
        "name": "Robin Roefs",
        "position": "GK",
        "age": 23,
        "ability": 76,
        "form": 76
      },
      {
        "name": "Nordi Mukiele",
        "position": "DEF",
        "age": 28,
        "ability": 78,
        "form": 80
      },
      {
        "name": "Omar Alderete",
        "position": "DEF",
        "age": 29,
        "ability": 76,
        "form": 75
      },
      {
        "name": "Daniel Ballard",
        "position": "DEF",
        "age": 26,
        "ability": 76,
        "form": 74
      },
      {
        "name": "Trai Hume",
        "position": "DEF",
        "age": 24,
        "ability": 73,
        "form": 70
      },
      {
        "name": "Enzo Le Fée",
        "position": "MID",
        "age": 26,
        "ability": 79,
        "form": 77
      },
      {
        "name": "Granit Xhaka",
        "position": "MID",
        "age": 33,
        "ability": 77,
        "form": 74
      },
      {
        "name": "Chemsdine Talbi",
        "position": "MID",
        "age": 21,
        "ability": 75,
        "form": 67
      },
      {
        "name": "Noah Sadiki",
        "position": "MID",
        "age": 21,
        "ability": 74,
        "form": 66
      },
      {
        "name": "Habib Diarra",
        "position": "MID",
        "age": 22,
        "ability": 76,
        "form": 68
      },
      {
        "name": "Brian Brobbey",
        "position": "FWD",
        "age": 24,
        "ability": 77,
        "form": 68
      },
      {
        "name": "Wilson Isidor",
        "position": "FWD",
        "age": 25,
        "ability": 75,
        "form": 62
      },
      {
        "name": "Reinildo Mandava",
        "position": "DEF",
        "age": 32,
        "ability": 73,
        "form": 68
      },
      {
        "name": "Chris Rigg",
        "position": "MID",
        "age": 19,
        "ability": 72,
        "form": 59
      },
      {
        "name": "Simon Adingra",
        "position": "MID",
        "age": 24,
        "ability": 73,
        "form": 61
      },
      {
        "name": "Luke O'Nien",
        "position": "DEF",
        "age": 31,
        "ability": 70,
        "form": 60
      },
      {
        "name": "Melker Ellborg",
        "position": "GK",
        "age": 23,
        "ability": 75,
        "form": 77
      },
      {
        "name": "Romaine Mundle",
        "position": "MID",
        "age": 23,
        "ability": 71,
        "form": 52
      }
    ]
  }
]

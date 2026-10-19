# Extension: Hybrid Values

I haven't implemented this, but this is how I'd approach it.

## Which datasets

The age and household composition tables are both useful, but for different things:
- Household composition has the bigger effect on day-to-day spending. 
- A household with two adults and two children spends about 1.31 times the average overall, and 1.39 times on food. 
- However, the table only has the 13 top-level groups and stops at group 13, so it says nothing about pensions, investments or mortgage capital (the 14.x codes).

- Age matters most for those items. 
- Pension contributions are £40.80 a week for households aged 30-49, but £1.80 for over-75s.

I'd use composition for groups 1-13 and age for 14.x.

## Combining them with the regional figures

- For each ONS code, I'd scale the regional figure by how much this type of household spends compared with the average household:

    estimate = regional value × (household's column ÷ all-households column)

- The regional table still provides the detail, and the other tables only provides the multiplier. For codes in groups 1-13 I'd use the ratio of the code's top-level group, so 4.3 (water) uses the ratio for group 4.
- Each code only gets one ratio. Multiplying the age and composition ratios together would count part of the same effect twice, since 30-49 households are also the ones most likely to have children.
- The ratios are worked out within each table rather than across tables, because the regional table covers a different period: its UK total is £571.80 a week, against £585.60 in the other two.

- The composition table has no all-households column, but row 14 gives the number of households in each column, so a weighted average gives one. It comes out at exactly the age table's all-households figures (£585.60 total, £61.90 food), which suggests the columns cover every household between them.

For the sample case (East Midlands, two clients aged 36 and 34, two dependents):

- Food uses the composition ratio, 85.9 ÷ 61.9 ≈ 1.39, so £265.20 a month becomes about £368.
- Pension uses the age ratio, 40.8 ÷ 28.6 ≈ 1.43, so £101.83 a month becomes £145.27.

## Working out the household from a case

- Adults: the number of clients.
- Children: `dependents`. Joint applicants list the same dependents, so I'd take the highest value rather than adding them up.
- Retired: if every client is retired. The table splits retired households by whether they mainly rely on the state pension, which the case doesn't tell us, so I'd merge those columns using their household counts.
- Age band: the oldest client's age. ONS goes by the highest earner in the household, and we don't have income, so the oldest client is the closest substitute.

## Limitations

- It assumes the household effect is the same in every region.
- Every code in a group gets the same ratio, so rent and water both go up by 1.14 for the sample case.
- Where a figure is missing I'd fall back to the parent category's ratio, or leave the regional figure as it is.
    - Small-sample figures can give extreme ratios on small groups like education, so the ratios might need capping (say between 0.5 and 2).

## Changes to the code

- The converter would need to read both sheets. 
- The age sheet works with the existing parser, but composition needs its own. 
- The service would build a household profile instead of just a region, and the estimator would apply one
ratio per code. 

-- Test slug generation function
SELECT 
    generate_slug('  Real Madrid  ') as test1,
    generate_slug('Barcelona/AC Milan') as test2,
    generate_slug('--Team Name--') as test3,
    generate_slug('Fenerbahçe vs Galatasaray') as test4,
    generate_slug('Man. City') as test5,
    generate_slug('  -  Extra  -  Spaces  -  ') as test6;
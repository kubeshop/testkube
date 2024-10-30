namespace ExampleProject;

public class Tests
{
    [SetUp]
    public void Setup()
    {
    }

    [Test]
    public void Test1()
    {
        Assert.Pass();
    }

    [Test]
    public void TestA() {
        Assert.That("A", Is.EqualTo("A"));
    }

    [Test]
    public void TestB() {
        Assert.That("B", Is.EqualTo("B"));
    }

    [Test]
    public void TestC() {
        Assert.That("C", Is.EqualTo("C"));
    }
}
